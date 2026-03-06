# Bank Mechanism Implementation Plan for Block-Fine-Tune

## Overview
This document outlines the implementation plan for adding a "bank" mechanism to the block-fine-tune project. The bank mechanism allows accounts to borrow from a shard-specific bank during migration, enabling payer transactions to proceed while accounts are migrating between shards.

## Goals
1. **Enable payer transactions during migration**: Allow accounts to "borrow" from bank to pay transactions while migrating
2. **Maintain system consistency**: Ensure atomicity of loan-migration operations
3. **Cross-shard coordination**: Handle loans taken in source shard and repaid in target shard
4. **Backward compatibility**: Maintain existing fine-tuned lock mechanism as fallback

## Architecture Design

### Core Components
1. **Bank Account**: Persistent account in each shard with initial capital
2. **Loan Manager**: Tracks active loans and repayments
3. **Transaction Aggregator**: Groups payer transactions into single loan
4. **Repayment Scheduler**: Creates repayment transactions after migration

### Data Flow
```
Account A migrating from S1 → S2:
1. A has pending payer transactions: A→B 100, A→C 50
2. Bank in S1 lends 150 to A (aggregated loan)
3. B and C receive payments immediately
4. A migrates to S2 with loan obligation
5. A repays 150 to bank in S2
6. Banks reconcile balances periodically
```

## Implementation Phases

### Phase 1: Foundation Setup

#### 1.1 Bank Data Structures
**Location**: `/bank/` new directory
**Files to create**:
- `bank.go` - Bank account and loan management
- `bank_state.go` - Bank state serialization
- `loan.go` - Loan record structure

**Key structures**:
```go
// In bank.go
type BankManager struct {
    ShardID        int
    BankAddress    string
    Balance        *big.Int
    Loans          map[string]*LoanRecord
    lock           sync.Mutex
}

type LoanRecord struct {
    LoanID         string
    Borrower       string
    Amount         *big.Int
    Interest       *big.Int
    SourceShard    int
    TargetShard    int
    Status         LoanStatus
    CreatedBlock   uint64
    DueBlock       uint64
    MigrationTxID  string
}

type LoanStatus int
const (
    LoanActive LoanStatus = iota
    LoanRepaid
    LoanDefaulted
)
```

#### 1.2 Configuration Updates
**Location**: `/params/config.go`
**Modifications**:
- Add bank-related configuration parameters
- Set default values for bank mechanism

```go
// Add to ChainConfig struct
EnableBankMechanism   bool
BankInitialBalance    *big.Int
BankInterestRate      *big.Int  // e.g., "1000000000000000000" for 0% (1e18 = 100%)
MaxLoanPerAccount     *big.Int
LoanRepaymentPeriod   uint64    // blocks until repayment due

// Update default config
Config = &ChainConfig{
    // ... existing fields
    EnableBankMechanism:   false,  // Disabled by default
    BankInitialBalance:    big.NewInt(1000000000000000000000000), // 1M tokens
    BankInterestRate:      big.NewInt(1000000000000000000),       // 0% interest
    MaxLoanPerAccount:     big.NewInt(1000000000000000000000),    // 1000 tokens
    LoanRepaymentPeriod:   100,    // 100 blocks to repay
}
```

#### 1.3 Genesis State Initialization
**Location**: `/chain/blockchain.go`
**Modifications**:
- Add bank accounts to genesis state
- Initialize bank balances

**Implementation steps**:
1. Modify `genesisStateTree()` function (line 625-629)
2. Create bank account for each shard with initial balance
3. Add bank addresses to `Account2Shard` and `AccountInOwnShard` mappings
4. Store bank address in params for easy access

### Phase 2: Transaction Processing Modifications

#### 2.1 Transaction Type Extensions
**Location**: `/core/transaction.go`
**Modifications**:
- Add bank-related fields to Transaction struct
- Add new transaction types if needed

```go
// Add to Transaction struct
IsBankLoan      bool    // Transaction represents bank loan
IsRepayment     bool    // Transaction is loan repayment
LoanID          string  // Reference to loan record
BankAddress     []byte  // Which bank is involved
```

#### 2.2 Transaction Pool Modifications
**Location**: `/core/txpool.go`
**Modifications**:
1. **Add loan aggregation logic** to `FetchTxs2Pack()`:
   - Track aggregated loan amounts per migrating account
   - Create bank loan transactions for aggregated amounts
   - Mark original transactions as "bank-processed"

2. **Add bank transaction handling**:
   - New method `ProcessBankLoans()` to handle loan creation
   - Modify locking logic to use bank mechanism when enabled

**Key changes in `FetchTxs2Pack()`**:
```go
// When account is locked and migrating:
if account.Lock_Acc[from] && !v.IsRelay && !v.Relay_Lock {
    if config.EnableBankMechanism {
        // Aggregate loan amount instead of locking
        loanAggregator[from] = new(big.Int).Add(loanAggregator[from], v.Value)
        v.IsBankLoan = true
        v.Success = true  // Payee receives immediately
        txs = append(txs, v)  // Include in current block
    } else {
        // Original behavior: lock transaction
        v.SenLock = true
        pool.Locking_TX_Pools[from] = append(pool.Locking_TX_Pools[from], v)
    }
}
```

#### 2.3 State Update Modifications
**Location**: `/chain/blockchain.go`
**Modifications**:
1. **Extend `getUpdatedTreeOfState()`**:
   - Add bank loan processing section
   - Handle bank balance updates
   - Process loan repayments

2. **Add helper functions**:
   - `processBankLoan()`: Deduct from bank, record loan
   - `processRepayment()`: Add to bank, mark loan repaid

**Implementation**:
```go
// In getUpdatedTreeOfState(), after regular transaction processing:
for _, tx := range txs {
    if tx.IsBankLoan {
        // Process bank loan
        bankState := getBankState(st, tx.BankAddress)
        bankState.Balance.Sub(bankState.Balance, tx.Value)
        st.Update(tx.BankAddress, bankState.Encode())
        
        // Record loan in bank manager
        bankManager.RecordLoan(tx.Recipient, tx.Value, tx.LoanID, 
                              params.ShardTable[config.ShardID], 
                              targetShardID)
    }
    
    if tx.IsRepayment {
        // Process repayment
        bankState := getBankState(st, tx.BankAddress)
        bankState.Balance.Add(bankState.Balance, tx.Value)
        st.Update(tx.BankAddress, bankState.Encode())
        
        // Mark loan as repaid
        bankManager.MarkRepaid(tx.LoanID)
    }
}
```

### Phase 3: Migration Integration 

#### 3.1 Migration Data Structure Updates
**Location**: `/pbft/PBFTforMigrate.go`
**Modifications**:
1. **Extend `BalancesAndPendings` struct**:
   - Add loan information field
   - Include source bank shard ID

2. **Modify `SendOut()` function**:
   - Include active loans in migration data
   - Track which bank provided loans

**Updated structures**:
```go
type BalancesAndPendings struct {
    // ... existing fields
    Loans map[string]*big.Int  // address -> loan amount
    SourceBankShard int        // Shard where loan was taken
    LoanIDs map[string]string // address -> loan ID
}

type BalanceAndPending struct {
    // ... existing fields
    LoanAmount *big.Int        // Loan amount for this account
    LoanID     string          // Associated loan ID
}
```

#### 3.2 Migration Completion Handling
**Location**: `/pbft/handleanns.go`
**Modifications**:
1. **Extend `handleAnns()` function**:
   - Schedule loan repayments after migration
   - Create repayment transactions to target shard's bank

2. **Add repayment scheduling logic**:
   - Calculate total repayment amount (principal + interest)
   - Create repayment transaction in TXns pool
   - Set repayment due block

**Implementation**:
```go
// After processing announcements, schedule repayments
func (p *Pbft) scheduleLoanRepayments(migrationData *BalancesAndPendings) {
    for addr, loanAmount := range migrationData.Loans {
        // Calculate repayment amount with interest
        repaymentAmount := calculateRepayment(loanAmount, config.BankInterestRate)
        
        // Create repayment transaction
        repaymentTx := &core.TXns{
            Address: addr,
            Change:  new(big.Int).Neg(repaymentAmount),  // Negative change = payment
            // ... other fields
        }
        
        // Add to TXns pool for execution
        p.Node.CurChain.TXns_pool.AddTXns(repaymentTx)
        
        // Record repayment obligation in target shard's bank
        targetBankManager.RecordRepayment(addr, repaymentAmount, 
                                         migrationData.LoanIDs[addr])
    }
}
```

#### 3.3 Cross-Shard Bank Communication
**Location**: New file `/bank/communication.go`
**Implementation**:
1. **Define message types**:
   - `BankLoanNotification`: Source shard notifies target about loan
   - `BankRepaymentConfirmation`: Target confirms repayment received
   - `BankBalanceReconciliation`: Periodic balance reconciliation

2. **Implement communication handlers**:
   - Handle incoming bank messages
   - Update local bank state based on cross-shard events

### Phase 4: Bank Management Module 

#### 4.1 Bank Manager Implementation
**Location**: `/bank/bank_manager.go`
**Features**:
1. **Loan management**:
   - Create new loans
   - Track loan status
   - Calculate interest
   - Handle defaults

2. **Risk management**:
   - Credit limits per account
   - Collateral requirements
   - Default probability estimation

3. **Balance management**:
   - Monitor bank reserves
   - Handle insufficient funds
   - Profit/loss tracking

**Key methods**:
```go
type BankManager interface {
    // Loan operations
    CreateLoan(borrower string, amount *big.Int, targetShard int) (*LoanRecord, error)
    ProcessRepayment(loanID string, amount *big.Int) error
    MarkDefault(loanID string) error
    
    // Risk management
    CheckCreditLimit(borrower string, requested *big.Int) (bool, error)
    CalculateCollateral(borrower string, amount *big.Int) (*big.Int, error)
    
    // Balance management
    GetAvailableBalance() *big.Int
    TransferToOtherBank(targetShard int, amount *big.Int) error
    ReconcileBalance() error
}
```

#### 4.2 Bank State Persistence
**Location**: `/bank/bank_state.go`
**Implementation**:
1. **Bank state serialization** for trie storage
2. **Loan record persistence** in separate database
3. **State snapshot and recovery** mechanisms

### Phase 5: Integration and Testing

#### 5.1 Integration Points
1. **Main initialization** (`/shard/shard.go`):
   - Initialize bank manager for each shard
   - Set up bank communication channels

2. **Blockchain integration** (`/chain/blockchain.go`):
   - Bank state updates during block processing
   - Loan-related transaction validation

3. **Consensus integration** (`/pbft/`):
   - Bank transactions in PBFT consensus
   - Cross-shard bank coordination during migration

#### 5.2 Testing Strategy
1. **Unit tests**:
   - Bank manager operations
   - Loan calculations
   - Transaction aggregation

2. **Integration tests**:
   - Complete migration with bank loans
   - Cross-shard repayment flow
   - Bank insolvency scenarios

3. **Performance tests**:
   - Bank mechanism overhead
   - Scalability with many concurrent loans
   - Comparison with fine-tuned lock only

**Test files to create**:
- `/bank/bank_test.go`
- `/test/test_bank_migration.go`
- `/test/test_bank_performance.go`

#### 5.3 Configuration and Deployment
1. **Configuration options**:
   - Enable/disable bank mechanism
   - Bank capital allocation
   - Interest rate settings
   - Risk parameters

2. **Deployment considerations**:
   - Genesis bank balances
   - Bank address generation
   - Upgrade path from existing system



Each phase includes:
- Code implementation
- Unit testing
- Integration with existing system
- Documentation updates
