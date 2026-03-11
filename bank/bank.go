package bank

import (
	"blockEmulator/params"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"sync"
	"time"
)

type BankManager struct {
	ShardID       int
	BankAddress   string
	Balance       *big.Int
	Loans         map[string]*LoanRecord
	Communication *BankCommunication
	lock          sync.Mutex
}

func NewBankManager(shardID int, initialBalance *big.Int) *BankManager {
	bankAddress := generateBankAddress(shardID)
	bm := &BankManager{
		ShardID:     shardID,
		BankAddress: bankAddress,
		Balance:     new(big.Int).Set(initialBalance),
		Loans:       make(map[string]*LoanRecord),
	}
	// Initialize bank communication
	bm.Communication = NewBankCommunication(shardID, bm)
	return bm
}

func generateBankAddress(shardID int) string {
	// Generate a deterministic bank address based on shard ID
	timestamp := time.Now().UnixNano()
	input := fmt.Sprintf("bank_shard_%d_%d", shardID, timestamp)
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:20]) // 20 bytes like Ethereum addresses
}

func (bm *BankManager) CreateLoan(borrower string, amount *big.Int, targetShard int) (*LoanRecord, error) {
	bm.lock.Lock()
	defer bm.lock.Unlock()

	// Check if bank has sufficient funds
	if bm.Balance.Cmp(amount) < 0 {
		return nil, fmt.Errorf("insufficient bank funds: have %s, need %s", bm.Balance.String(), amount.String())
	}

	// Check borrower credit limit from config
	maxLoan := params.Config.MaxLoanPerAccount
	if maxLoan != nil && amount.Cmp(maxLoan) > 0 {
		return nil, fmt.Errorf("loan amount %s exceeds maximum per account %s", amount.String(), maxLoan.String())
	}

	// Generate loan ID
	loanID := generateLoanID(borrower, bm.ShardID, targetShard)

	// Create loan record
	loan := NewLoanRecord(loanID, borrower, amount, bm.ShardID, targetShard)
	loan.CreatedBlock = getCurrentBlockHeight()
	loan.DueBlock = loan.CreatedBlock + params.Config.LoanRepaymentPeriod

	// Deduct from bank balance
	bm.Balance.Sub(bm.Balance, amount)

	// Store loan
	bm.Loans[loanID] = loan

	log.Printf("Bank Shard %d: Created loan %s to %s for amount %s", bm.ShardID, loanID, borrower, amount.String())
	return loan, nil
}

func (bm *BankManager) ProcessRepayment(loanID string, amount *big.Int) error {
	bm.lock.Lock()
	defer bm.lock.Unlock()

	loan, exists := bm.Loans[loanID]
	if !exists {
		return fmt.Errorf("loan %s not found", loanID)
	}

	if !loan.IsActive() {
		return fmt.Errorf("loan %s is not active (status: %d)", loanID, loan.Status)
	}

	// Add repayment to bank balance
	bm.Balance.Add(bm.Balance, amount)

	// Mark loan as repaid
	loan.MarkRepaid()

	log.Printf("Bank Shard %d: Processed repayment for loan %s, amount %s", bm.ShardID, loanID, amount.String())
	return nil
}

func (bm *BankManager) GetLoan(loanID string) (*LoanRecord, bool) {
	bm.lock.Lock()
	defer bm.lock.Unlock()

	loan, exists := bm.Loans[loanID]
	return loan, exists
}

func (bm *BankManager) GetActiveLoans() []*LoanRecord {
	bm.lock.Lock()
	defer bm.lock.Unlock()

	var activeLoans []*LoanRecord
	for _, loan := range bm.Loans {
		if loan.IsActive() {
			activeLoans = append(activeLoans, loan)
		}
	}
	return activeLoans
}

func (bm *BankManager) GetAvailableBalance() *big.Int {
	bm.lock.Lock()
	defer bm.lock.Unlock()
	return new(big.Int).Set(bm.Balance)
}

func (bm *BankManager) GetTotalLoansOutstanding() *big.Int {
	bm.lock.Lock()
	defer bm.lock.Unlock()

	total := big.NewInt(0)
	for _, loan := range bm.Loans {
		if loan.IsActive() {
			total.Add(total, loan.Amount)
		}
	}
	return total
}

func generateLoanID(borrower string, sourceShard, targetShard int) string {
	timestamp := time.Now().UnixNano()
	input := fmt.Sprintf("loan_%s_%d_%d_%d", borrower, sourceShard, targetShard, timestamp)
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:16])
}

func getCurrentBlockHeight() uint64 {
	// This is a placeholder - actual implementation would get from blockchain
	// For now, return 0 and we'll implement properly later
	return 0
}

// Helper function to get shard ID from shard string (e.g., "S0" -> 0)
func GetShardIDFromString(shardStr string) (int, error) {
	if len(shardStr) < 2 || shardStr[0] != 'S' {
		return 0, fmt.Errorf("invalid shard string format: %s", shardStr)
	}
	shardNum, err := strconv.Atoi(shardStr[1:])
	if err != nil {
		return 0, fmt.Errorf("invalid shard number in %s: %v", shardStr, err)
	}
	return shardNum, nil
}

// Helper function to get bank address for a shard
func GetBankAddressForShard(shardID int) string {
	// This would be called after bank is initialized
	// For now, generate a deterministic address
	timestamp := int64(0) // Use 0 for deterministic generation
	input := fmt.Sprintf("bank_shard_%d_%d", shardID, timestamp)
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:20])
}

// ScheduleLoanRepayments schedules loan repayments for migrated accounts
// This function is a placeholder and will be integrated into migration completion handling.
func ScheduleLoanRepayments(migrationData map[string]*big.Int, borrower string, targetShard int) (*big.Int, error) {
	// Placeholder implementation: returns the sum of loan amounts in migrationData
	if migrationData == nil {
		return big.NewInt(0), nil
	}

	totalRepayment := big.NewInt(0)
	for _, amount := range migrationData {
		totalRepayment.Add(totalRepayment, amount)
	}

	return totalRepayment, nil
}

// GetLoanInfoForAccount returns loan information for a migrated account
// This function is a placeholder and will be integrated into migration data preparation.
func GetLoanInfoForAccount(borrower string, sourceShard int) (map[string]*big.Int, map[string]string, error) {
	// Placeholder implementation: returns empty maps for now.
	// Actual implementation would query the BankManager for active loans of the borrower
	// in the source shard.
	return make(map[string]*big.Int), make(map[string]string), nil
}

// RecordIncomingLoan records a loan that was taken in another shard and needs to be repaid locally
func RecordIncomingLoan(borrower string, amount *big.Int, loanID string, sourceShard int, targetShard int) error {
	// This is a placeholder implementation
	// In a full implementation, this would:
	// 1. Create a loan record in the local bank
	// 2. Mark it as "cross-shard" loan
	// 3. Schedule it for repayment tracking

	fmt.Printf("Recording incoming loan: borrower=%s, amount=%s, loanID=%s, sourceShard=%d, targetShard=%d\n",
		borrower, amount.String(), loanID, sourceShard, targetShard)

	// For now, just log the information
	// Actual implementation would use BankManager to store the loan record
	return nil
}
