package bank

import (
	"math/big"
)

type LoanStatus int

const (
	LoanActive LoanStatus = iota
	LoanRepaid
	LoanDefaulted
)

type LoanRecord struct {
	LoanID        string
	Borrower      string
	Amount        *big.Int
	Interest      *big.Int
	SourceShard   int
	TargetShard   int
	Status        LoanStatus
	CreatedBlock  uint64
	DueBlock      uint64
	MigrationTxID string
}

func NewLoanRecord(loanID, borrower string, amount *big.Int, sourceShard, targetShard int) *LoanRecord {
	return &LoanRecord{
		LoanID:       loanID,
		Borrower:     borrower,
		Amount:       new(big.Int).Set(amount),
		Interest:     big.NewInt(0),
		SourceShard:  sourceShard,
		TargetShard:  targetShard,
		Status:       LoanActive,
		CreatedBlock: 0,
		DueBlock:     0,
	}
}

func (lr *LoanRecord) MarkRepaid() {
	lr.Status = LoanRepaid
}

func (lr *LoanRecord) MarkDefaulted() {
	lr.Status = LoanDefaulted
}

func (lr *LoanRecord) IsActive() bool {
	return lr.Status == LoanActive
}

func (lr *LoanRecord) IsRepaid() bool {
	return lr.Status == LoanRepaid
}

func (lr *LoanRecord) IsDefaulted() bool {
	return lr.Status == LoanDefaulted
}
