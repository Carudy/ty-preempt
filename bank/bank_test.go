package bank

import (
	"math/big"
	"testing"
)

func TestBankManagerCreation(t *testing.T) {
	initialBalance := big.NewInt(1000000)
	bm := NewBankManager(1, initialBalance)

	if bm.ShardID != 1 {
		t.Errorf("Expected ShardID 1, got %d", bm.ShardID)
	}

	if bm.Balance.Cmp(initialBalance) != 0 {
		t.Errorf("Expected balance %s, got %s", initialBalance.String(), bm.Balance.String())
	}

	if bm.BankAddress == "" {
		t.Error("Bank address should not be empty")
	}

	if len(bm.Loans) != 0 {
		t.Errorf("Expected empty loans map, got %d loans", len(bm.Loans))
	}
}

func TestLoanCreation(t *testing.T) {
	initialBalance := big.NewInt(1000000)
	bm := NewBankManager(1, initialBalance)

	loanAmount := big.NewInt(100)
	loan, err := bm.CreateLoan("test_borrower", loanAmount, 2)

	if err != nil {
		t.Errorf("Failed to create loan: %v", err)
	}

	if loan == nil {
		t.Error("Loan should not be nil")
	}

	if loan.Borrower != "test_borrower" {
		t.Errorf("Expected borrower 'test_borrower', got '%s'", loan.Borrower)
	}

	if loan.Amount.Cmp(loanAmount) != 0 {
		t.Errorf("Expected loan amount %s, got %s", loanAmount.String(), loan.Amount.String())
	}

	if loan.SourceShard != 1 {
		t.Errorf("Expected source shard 1, got %d", loan.SourceShard)
	}

	if loan.TargetShard != 2 {
		t.Errorf("Expected target shard 2, got %d", loan.TargetShard)
	}

	if !loan.IsActive() {
		t.Error("New loan should be active")
	}

	// Check bank balance was reduced
	expectedBalance := big.NewInt(999900) // 1,000,000 - 100
	if bm.Balance.Cmp(expectedBalance) != 0 {
		t.Errorf("Expected bank balance %s after loan, got %s", expectedBalance.String(), bm.Balance.String())
	}

	// Check loan is stored
	retrievedLoan, exists := bm.GetLoan(loan.LoanID)
	if !exists {
		t.Error("Loan should be retrievable after creation")
	}
	if retrievedLoan.LoanID != loan.LoanID {
		t.Errorf("Retrieved loan ID mismatch: expected %s, got %s", loan.LoanID, retrievedLoan.LoanID)
	}
}

func TestLoanCreationInsufficientFunds(t *testing.T) {
	initialBalance := big.NewInt(100)
	bm := NewBankManager(1, initialBalance)

	loanAmount := big.NewInt(200) // More than bank has
	loan, err := bm.CreateLoan("test_borrower", loanAmount, 2)

	if err == nil {
		t.Error("Expected error for insufficient funds")
	}

	if loan != nil {
		t.Error("Loan should be nil when creation fails")
	}

	// Bank balance should remain unchanged
	if bm.Balance.Cmp(initialBalance) != 0 {
		t.Errorf("Bank balance should remain %s, got %s", initialBalance.String(), bm.Balance.String())
	}
}

func TestLoanRepayment(t *testing.T) {
	initialBalance := big.NewInt(1000000)
	bm := NewBankManager(1, initialBalance)

	loanAmount := big.NewInt(100)
	loan, err := bm.CreateLoan("test_borrower", loanAmount, 2)
	if err != nil {
		t.Fatalf("Failed to create loan: %v", err)
	}

	// Process repayment
	repaymentAmount := big.NewInt(100)
	err = bm.ProcessRepayment(loan.LoanID, repaymentAmount)
	if err != nil {
		t.Errorf("Failed to process repayment: %v", err)
	}

	// Check loan status
	updatedLoan, exists := bm.GetLoan(loan.LoanID)
	if !exists {
		t.Error("Loan should still exist after repayment")
	}

	if !updatedLoan.IsRepaid() {
		t.Error("Loan should be marked as repaid")
	}

	if updatedLoan.IsActive() {
		t.Error("Loan should not be active after repayment")
	}

	// Check bank balance was restored
	if bm.Balance.Cmp(initialBalance) != 0 {
		t.Errorf("Expected bank balance %s after repayment, got %s", initialBalance.String(), bm.Balance.String())
	}
}

func TestLoanRepaymentNotFound(t *testing.T) {
	initialBalance := big.NewInt(1000000)
	bm := NewBankManager(1, initialBalance)

	repaymentAmount := big.NewInt(100)
	err := bm.ProcessRepayment("non_existent_loan", repaymentAmount)

	if err == nil {
		t.Error("Expected error for non-existent loan")
	}
}

func TestLoanRepaymentAlreadyRepaid(t *testing.T) {
	initialBalance := big.NewInt(1000000)
	bm := NewBankManager(1, initialBalance)

	loanAmount := big.NewInt(100)
	loan, err := bm.CreateLoan("test_borrower", loanAmount, 2)
	if err != nil {
		t.Fatalf("Failed to create loan: %v", err)
	}

	// First repayment should succeed
	err = bm.ProcessRepayment(loan.LoanID, loanAmount)
	if err != nil {
		t.Fatalf("First repayment failed: %v", err)
	}

	// Second repayment should fail
	err = bm.ProcessRepayment(loan.LoanID, loanAmount)
	if err == nil {
		t.Error("Expected error for already repaid loan")
	}
}

func TestGetActiveLoans(t *testing.T) {
	initialBalance := big.NewInt(1000000)
	bm := NewBankManager(1, initialBalance)

	// Create multiple loans
	loan1, err := bm.CreateLoan("borrower1", big.NewInt(100), 2)
	if err != nil {
		t.Fatalf("Failed to create loan1: %v", err)
	}

	loan2, err := bm.CreateLoan("borrower2", big.NewInt(200), 3)
	if err != nil {
		t.Fatalf("Failed to create loan2: %v", err)
	}

	// Repay one loan
	err = bm.ProcessRepayment(loan1.LoanID, big.NewInt(100))
	if err != nil {
		t.Fatalf("Failed to repay loan1: %v", err)
	}

	activeLoans := bm.GetActiveLoans()
	if len(activeLoans) != 1 {
		t.Errorf("Expected 1 active loan, got %d", len(activeLoans))
	}

	if activeLoans[0].LoanID != loan2.LoanID {
		t.Errorf("Expected active loan ID %s, got %s", loan2.LoanID, activeLoans[0].LoanID)
	}
}

func TestGetAvailableBalance(t *testing.T) {
	initialBalance := big.NewInt(1000000)
	bm := NewBankManager(1, initialBalance)

	availableBalance := bm.GetAvailableBalance()
	if availableBalance.Cmp(initialBalance) != 0 {
		t.Errorf("Expected available balance %s, got %s", initialBalance.String(), availableBalance.String())
	}

	// Create a loan
	_, err := bm.CreateLoan("test_borrower", big.NewInt(300), 2)
	if err != nil {
		t.Fatalf("Failed to create loan: %v", err)
	}

	// Available balance should be reduced
	availableBalance = bm.GetAvailableBalance()
	expectedBalance := big.NewInt(999700) // 1,000,000 - 300
	if availableBalance.Cmp(expectedBalance) != 0 {
		t.Errorf("Expected available balance %s after loan, got %s", expectedBalance.String(), availableBalance.String())
	}
}

func TestGetTotalLoansOutstanding(t *testing.T) {
	initialBalance := big.NewInt(1000000)
	bm := NewBankManager(1, initialBalance)

	// Initially no loans
	totalOutstanding := bm.GetTotalLoansOutstanding()
	if totalOutstanding.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Expected 0 outstanding loans, got %s", totalOutstanding.String())
	}

	// Create loans
	_, err := bm.CreateLoan("borrower1", big.NewInt(100), 2)
	if err != nil {
		t.Fatalf("Failed to create loan1: %v", err)
	}

	_, err = bm.CreateLoan("borrower2", big.NewInt(200), 3)
	if err != nil {
		t.Fatalf("Failed to create loan2: %v", err)
	}

	// Total should be 300
	totalOutstanding = bm.GetTotalLoansOutstanding()
	expectedTotal := big.NewInt(300)
	if totalOutstanding.Cmp(expectedTotal) != 0 {
		t.Errorf("Expected total outstanding %s, got %s", expectedTotal.String(), totalOutstanding.String())
	}

	// Repay one loan - find loan by iterating through active loans
	var loan1 *LoanRecord
	activeLoans := bm.GetActiveLoans()
	for _, loan := range activeLoans {
		if loan.Borrower == "borrower1" {
			loan1 = loan
			break
		}
	}
	if loan1 != nil {
		bm.ProcessRepayment(loan1.LoanID, big.NewInt(100))
	}

	// Total should now be 200
	totalOutstanding = bm.GetTotalLoansOutstanding()
	expectedTotal = big.NewInt(200)
	if totalOutstanding.Cmp(expectedTotal) != 0 {
		t.Errorf("Expected total outstanding %s after repayment, got %s", expectedTotal.String(), totalOutstanding.String())
	}
}

func TestBankStateSerialization(t *testing.T) {
	bs := NewBankState(1, big.NewInt(1000))
	encoded := bs.Encode()
	decoded := DecodeBankState(encoded)

	if decoded.ShardID != bs.ShardID {
		t.Errorf("Expected ShardID %d after decode, got %d", bs.ShardID, decoded.ShardID)
	}

	if decoded.Balance.Cmp(bs.Balance) != 0 {
		t.Errorf("Expected balance %s after decode, got %s", bs.Balance.String(), decoded.Balance.String())
	}
}

func TestBankStateCanLend(t *testing.T) {
	bs := NewBankState(1, big.NewInt(1000))

	if !bs.CanLend(big.NewInt(500)) {
		t.Error("Should be able to lend 500 when balance is 1000")
	}

	if !bs.CanLend(big.NewInt(1000)) {
		t.Error("Should be able to lend 1000 when balance is 1000")
	}

	if bs.CanLend(big.NewInt(1001)) {
		t.Error("Should not be able to lend 1001 when balance is 1000")
	}
}

func TestBankStateLend(t *testing.T) {
	bs := NewBankState(1, big.NewInt(1000))

	// Successful lend
	if !bs.Lend(big.NewInt(500)) {
		t.Error("Lend should succeed for amount within balance")
	}

	if bs.Balance.Cmp(big.NewInt(500)) != 0 {
		t.Errorf("Expected balance 500 after lending 500, got %s", bs.Balance.String())
	}

	// Failed lend (insufficient funds)
	if bs.Lend(big.NewInt(600)) {
		t.Error("Lend should fail for amount exceeding balance")
	}

	// Balance should remain unchanged after failed lend
	if bs.Balance.Cmp(big.NewInt(500)) != 0 {
		t.Errorf("Expected balance 500 after failed lend, got %s", bs.Balance.String())
	}
}

func TestBankStateReceiveRepayment(t *testing.T) {
	bs := NewBankState(1, big.NewInt(1000))

	bs.Lend(big.NewInt(300))
	bs.ReceiveRepayment(big.NewInt(300))

	if bs.Balance.Cmp(big.NewInt(1000)) != 0 {
		t.Errorf("Expected balance 1000 after repayment, got %s", bs.Balance.String())
	}
}

func TestGetShardIDFromString(t *testing.T) {
	testCases := []struct {
		input     string
		expected  int
		shouldErr bool
	}{
		{"S0", 0, false},
		{"S1", 1, false},
		{"S10", 10, false},
		{"S123", 123, false},
		{"s0", 0, true},   // lowercase s
		{"0", 0, true},    // missing S prefix
		{"S", 0, true},    // missing number
		{"Sabc", 0, true}, // non-numeric
	}

	for _, tc := range testCases {
		result, err := GetShardIDFromString(tc.input)

		if tc.shouldErr {
			if err == nil {
				t.Errorf("Expected error for input %s, got none", tc.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input %s: %v", tc.input, err)
			}
			if result != tc.expected {
				t.Errorf("For input %s: expected %d, got %d", tc.input, tc.expected, result)
			}
		}
	}
}

func TestGetBankAddressForShard(t *testing.T) {
	address1 := GetBankAddressForShard(0)
	address2 := GetBankAddressForShard(1)
	address3 := GetBankAddressForShard(0) // Same shard should give same address

	if address1 == "" {
		t.Error("Bank address should not be empty")
	}

	if address2 == "" {
		t.Error("Bank address should not be empty")
	}

	if address1 == address2 {
		t.Error("Different shards should have different bank addresses")
	}

	if address1 != address3 {
		t.Error("Same shard should have same deterministic bank address")
	}
}
