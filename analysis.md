# Analysis of Block-Fine-Tune Project (Fine-tune-lock Branch)

## Project Overview
This project implements a sharded blockchain system with account migration capabilities, specifically featuring the "Fine-tuned Lock" mechanism described in the INFOCOM'24 paper "Account Migration across Blockchain Shards using Fine-tuned Lock Mechanism."

## Key Directories and Files

### Core Structure
- `/main.go` - Entry point, runs test_shard()
- `/shard/shard.go` - Node initialization and transaction loading
- `/chain/blockchain.go` - Blockchain operations (AddBlock, GenerateBlock, state updates)
- `/core/` - Transaction types and pools
- `/account/` - Account state and mapping management
- `/params/config.go` - Configuration parameters
- `/pbft/` - Consensus and migration coordination
- `/utils/utils.go` - Utility functions
- `/test/` - Test files

### Transaction Types (in /core/)
- `transaction.go` - Standard transactions
- `txmig1.go` - Migration request (TXmig1: account to migrate)
- `txmig2.go` - Migration completion (TXmig2: migrated account with state)
- `txann.go` - Announcement (TXann: migration announced)
- `txns.go` - Balance change (TXns: plus/minus transactions)
- `txrelay.go` - Relay transactions

### Transaction Pools (in /core/)
- `txpool.go` - Main transaction pool with locking mechanisms
- `txmig1pool.go` - Migration request pool
- `txmig2pool.go` - Migration completion pool
- `txannpool.go` - Announcement pool
- `txnspool.go` - Balance change pool

## Key Mechanisms

### 1. Account Migration Flow
1. **TXmig1**: Account marked for migration (source shard)
2. **Account Locking**: Account locked in source shard (if Lock_Acc_When_Migrating)
3. **State Transfer**: Account state sent to target shard via SendOut()
4. **TXmig2**: Account added to target shard state
5. **TXann**: Announcement sent back to source shard
6. **TXns**: Balance changes calculated and sent to target shard
7. **Account Unlocking**: Account unlocked in source shard

### 2. Fine-tuned Lock Mechanism
**Configuration**: `params.Config.RelayLock` (boolean)

#### Traditional Lock (RelayLock = false):
- Both payer (sender) and payee (recipient) transactions involving locked accounts are blocked
- Transactions moved to Locking_TX_Pools

#### Fine-tuned Lock (RelayLock = true):
- **Payer TX**: Locked (cannot deduct from locked sender)
- **Payee TX**: Can execute (can add to locked recipient)
- Creates relay-lock copies (Relay_Lock = true) for tracking without blocking execution

#### Key Code Locations:

1. **Configuration** (`/params/config.go#L120-124`):
   ```go
   RelayLock:                false,  // Set to true for fine-tuned lock
   ```

2. **Transaction Pool Decision Logic** (`/core/txpool.go#L206-240`):
   ```go
   } else if account.Lock_Acc[to] {
       if !config.RelayLock {
           // Traditional lock: move transaction to Locking_TX_Pools
           pool.Locking_TX_Pools[to] = append(pool.Locking_TX_Pools[to], v)
           continue
       } else {
           // Fine-tuned lock: create copy with Relay_Lock flag
           decoded.Relay_Lock = true
           pool.Locking_TX_Pools[to] = append(pool.Locking_TX_Pools[to], decoded)
           // Original transaction continues processing
       }
   }
   ```

3. **State Update Logic** (`/chain/blockchain.go#L370-380`):
   ```go
   if !tx.IsRelay && !tx.Relay_Lock {
       // Only deduct from sender if NOT a Relay_Lock transaction
       account_state.Balance.Sub(account_state.Balance, tx.Value)
       st.Update(tx.Sender, account_state.Encode())
   }
   ```

4. **Transaction Flags** (`/core/transaction.go#L15-37`):
   ```go
   type Transaction struct {
       // ... other fields
       IsRelay    bool  // Is this a relay transaction?
       SenLock    bool  // Is sender locked?
       RecLock    bool  // Is recipient locked?
       Relay_Lock bool  // Is this a relay-lock transaction? (fine-tuned lock)
   }
   ```

5. **Lock Checking Logic** (`/core/txpool.go#L192-204`):
   ```go
   if account.Lock_Acc[from] && !v.IsRelay && !v.Relay_Lock {
       // Payer transaction: always locked regardless of RelayLock setting
       v.SenLock = true
       pool.Locking_TX_Pools[from] = append(pool.Locking_TX_Pools[from], v)
       continue
   }
   ```

6. **Migration Completion Handling** (`/pbft/handleanns.go#L123-140`):
   ```go
   // When migration completes, handle locked transactions
   if hex.EncodeToString(tx.Recipient) == ann.Address && !tx.IsRelay && !tx.Relay_Lock {
       // Payee transactions can be released
       tx.Success = true
       p.Node.CurChain.Tx_pool.Queue = append(p.Node.CurChain.Tx_pool.Queue, tx)
   }
   ```

### 3. Transaction Processing Pipeline
1. **Injection**: Transactions injected into pool via InjectTxs2Shard() or NewInjectTxs2Shard()
2. **Fetching**: FetchTxs2Pack() retrieves transactions for block inclusion
3. **Lock Checking**: Checks if sender/recipient are locked/migrating
4. **Block Generation**: GenerateBlock() collects transactions and migration data
5. **State Update**: getUpdatedTreeOfState() applies transaction effects
6. **Block Addition**: AddBlock() finalizes block and updates account mappings

### 4. Inter-Shard Communication
- **SendOut()** (PBFTforMigrate.go): Sends migrating accounts to target shards
- **handleBalancesAndPendings()**: Processes incoming migration data
- **TrySendChangesAndPendings()**: Sends balance changes after announcement
- **TCP Communication**: utils.TcpDial() for shard-to-shard communication

### 5. Account State Management
- **Account2Shard**: Global map of account to shard mapping
- **AccountInOwnShard**: Per-shard map of local accounts
- **Lock_Acc**: Set of locked accounts (during migration)
- **BalanceBeforeOut**: Tracks account balances at migration start for TXns calculation

## Configuration Parameters (Key Flags)

### Migration Control
- `Lock_Acc_When_Migrating`: Whether to lock accounts during migration
- `RelayLock`: Enable fine-tuned lock mechanism (payee transactions can execute)
- `Stop_When_Migrating`: Whether to stop normal processing during migration
- `Not_Lock_immediately`: Delay locking until specific block height

### Experiment Modes
- `Bu_Tong_Bi_Li`: Different transaction ratio experiments
- `Bu_Tong_Shi_Jian`: Different migration timing experiments
- `Fail`: Migration failure experiments
- `Cross_Chain`: Cross-chain migration experiments
- `Pressure`: Stress testing with multiple accounts

### Performance Parameters
- `MaxBlockSize`: Maximum transactions per block
- `MaxMigSize`: Maximum migration transactions per block
- `Block_interval`: Time between blocks
- `Inject_speed`: Transaction injection rate

## Data Structures

### AccountState
```go
type AccountState struct {
    Balance  *big.Int  // Account balance
    Migrate  int       // Migration status (-1 = not migrating, shardID = migrating to)
    Location int       // Current shard location
}
```

### Transaction
```go
type Transaction struct {
    Sender, Recipient []byte
    Value             *big.Int
    Id                int
    IsRelay           bool     // Is this a relay transaction?
    SenLock           bool     // Is sender locked?
    RecLock           bool     // Is recipient locked?
    Relay_Lock        bool     // Is this a relay-lock transaction?
    // Timing fields for metrics
    RequestTime, CommitTime, LockTime, UnlockTime int64
}
```

## Migration Coordination Flow

1. **Source Shard Leader (N0)**:
   - Detects accounts to migrate (TXmig1 in block)
   - Calls SendOut() after PBFT commit
   - Sends BalancesAndPendings to target shard

2. **Target Shard Leader (N0)**:
   - Receives migration data via handleBalancesAndPendings()
   - Initiates intra-shard consensus for incoming accounts
   - Adds accounts via TXmig2

3. **Announcement Phase**:
   - Source shard sends TXann to confirm migration
   - Calculates balance changes (TXns) for period between TXmig1 and TXann
   - Sends changes to target shard
   - Unlocks account in source shard

## Performance Considerations

### State Trie Management
- Uses Ethereum's Merkle Patricia Trie for state storage
- getUpdatedTreeOfState() updates trie in memory, then commits to disk
- Separate transaction trees for standard and migration transactions

### Locking Overhead
- Multiple lock types for different resources:
  - Account2ShardLock: Global account mapping
  - Lock_Acc_Lock: Account locking status
  - Outing_Acc_Before_Announce_Lock: Pre-announcement migration status
  - Pool locks for transaction pools

### Metrics Collection
- Block timing logs (blocktimelog)
- Queue length logs (queuelenlog)
- Epoch change timing (epochChangelog)
- Transaction timing metrics (request, commit, lock, unlock times)

## Testing and Experimentation

The system supports multiple experiment modes through configuration flags:
- Different migration timing scenarios
- Failure recovery testing
- Cross-chain migration simulations
- Pressure testing with multiple concurrent migrations
- Transaction ratio experiments for performance analysis

## Key Insights

1. **Fine-tuned Lock reduces migration impact** by allowing payee transactions to proceed while blocking payer transactions
2. **Two-phase migration protocol** ensures consistency across shards
3. **Balance tracking** handles transactions that occur during migration window
4. **Flexible configuration** supports various research experiments
5. **PBFT consensus** ensures agreement within shards during migration operations

This implementation provides a research platform for studying account migration in sharded blockchains with configurable locking strategies.
