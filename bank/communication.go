package bank

import (
	"blockEmulator/params"
	"blockEmulator/utils"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"sync"
	"time"
)

// BankMessageType represents the type of bank-to-bank message
type BankMessageType string

const (
	// BankLoanNotification notifies target shard about a loan taken by a migrating account
	BankLoanNotification BankMessageType = "bank_loan_notification"

	// BankRepaymentConfirmation confirms that a repayment was received
	BankRepaymentConfirmation BankMessageType = "bank_repayment_confirmation"

	// BankBalanceReconciliation initiates balance reconciliation between banks
	BankBalanceReconciliation BankMessageType = "bank_balance_reconciliation"

	// BankLoanTransfer transfers loan ownership between banks
	BankLoanTransfer BankMessageType = "bank_loan_transfer"
)

// BankMessage represents a message between banks in different shards
type BankMessage struct {
	Type      BankMessageType `json:"type"`
	Timestamp int64           `json:"timestamp"`
	Sender    int             `json:"sender_shard"`   // Source shard ID
	Receiver  int             `json:"receiver_shard"` // Target shard ID
	Content   json.RawMessage `json:"content"`        // Message-specific content
}

// BankLoanNotificationContent contains loan information for notification
type BankLoanNotificationContent struct {
	Borrower    string   `json:"borrower"`
	LoanID      string   `json:"loan_id"`
	Amount      *big.Int `json:"amount"`
	Interest    *big.Int `json:"interest"`
	CreatedTime int64    `json:"created_time"`
	DueTime     int64    `json:"due_time"`
	MigrationTx string   `json:"migration_tx"` // Migration transaction ID
}

// BankRepaymentConfirmationContent confirms repayment receipt
type BankRepaymentConfirmationContent struct {
	LoanID        string   `json:"loan_id"`
	RepaymentTx   string   `json:"repayment_tx"`
	Amount        *big.Int `json:"amount"`
	RepaymentTime int64    `json:"repayment_time"`
	Borrower      string   `json:"borrower"`
}

// BankBalanceReconciliationContent contains balance reconciliation data
type BankBalanceReconciliationContent struct {
	PeriodStart      int64           `json:"period_start"`
	PeriodEnd        int64           `json:"period_end"`
	NetBalance       *big.Int        `json:"net_balance"` // Positive = target owes source
	Transactions     []*LoanTransfer `json:"transactions"`
	ReconciliationID string          `json:"reconciliation_id"`
}

// BankLoanTransferContent transfers loan ownership
type BankLoanTransferContent struct {
	LoanID       string   `json:"loan_id"`
	Borrower     string   `json:"borrower"`
	Amount       *big.Int `json:"amount"`
	Interest     *big.Int `json:"interest"`
	TransferTime int64    `json:"transfer_time"`
}

// LoanTransfer represents a single loan transfer between banks
type LoanTransfer struct {
	LoanID    string   `json:"loan_id"`
	Borrower  string   `json:"borrower"`
	Amount    *big.Int `json:"amount"`
	FromShard int      `json:"from_shard"`
	ToShard   int      `json:"to_shard"`
	Time      int64    `json:"time"`
}

// BankCommunication handles cross-shard bank communication
type BankCommunication struct {
	ShardID      int
	BankManager  *BankManager
	pendingLoans map[string]*PendingLoan // loanID -> pending loan
	lock         sync.Mutex
}

// PendingLoan tracks loans awaiting confirmation
type PendingLoan struct {
	LoanID      string
	Borrower    string
	Amount      *big.Int
	SourceShard int
	TargetShard int
	CreatedTime int64
	Status      PendingLoanStatus
}

// PendingLoanStatus represents the status of a pending loan
type PendingLoanStatus string

const (
	PendingLoanNotified    PendingLoanStatus = "notified"
	PendingLoanTransferred PendingLoanStatus = "transferred"
	PendingLoanConfirmed   PendingLoanStatus = "confirmed"
)

// NewBankCommunication creates a new BankCommunication instance
func NewBankCommunication(shardID int, bankManager *BankManager) *BankCommunication {
	return &BankCommunication{
		ShardID:      shardID,
		BankManager:  bankManager,
		pendingLoans: make(map[string]*PendingLoan),
	}
}

// NotifyTargetShard sends loan notification to target shard
func (bc *BankCommunication) NotifyTargetShard(borrower string, loanID string, amount *big.Int,
	interest *big.Int, targetShard int, migrationTx string) error {

	content := BankLoanNotificationContent{
		Borrower:    borrower,
		LoanID:      loanID,
		Amount:      new(big.Int).Set(amount),
		Interest:    new(big.Int).Set(interest),
		CreatedTime: time.Now().Unix(),
		DueTime:     time.Now().Add(time.Duration(params.Config.LoanRepaymentPeriod) * time.Second).Unix(),
		MigrationTx: migrationTx,
	}

	contentBytes, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("failed to marshal loan notification: %v", err)
	}

	message := BankMessage{
		Type:      BankLoanNotification,
		Timestamp: time.Now().Unix(),
		Sender:    bc.ShardID,
		Receiver:  targetShard,
		Content:   contentBytes,
	}

	// Record as pending loan
	bc.lock.Lock()
	bc.pendingLoans[loanID] = &PendingLoan{
		LoanID:      loanID,
		Borrower:    borrower,
		Amount:      new(big.Int).Set(amount),
		SourceShard: bc.ShardID,
		TargetShard: targetShard,
		CreatedTime: time.Now().Unix(),
		Status:      PendingLoanNotified,
	}
	bc.lock.Unlock()

	// Send message to target shard's bank
	return bc.sendMessageToShard(targetShard, message)
}

// ConfirmRepayment sends repayment confirmation to source shard
func (bc *BankCommunication) ConfirmRepayment(loanID string, repaymentTx string,
	amount *big.Int, borrower string, sourceShard int) error {

	content := BankRepaymentConfirmationContent{
		LoanID:        loanID,
		RepaymentTx:   repaymentTx,
		Amount:        new(big.Int).Set(amount),
		RepaymentTime: time.Now().Unix(),
		Borrower:      borrower,
	}

	contentBytes, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("failed to marshal repayment confirmation: %v", err)
	}

	message := BankMessage{
		Type:      BankRepaymentConfirmation,
		Timestamp: time.Now().Unix(),
		Sender:    bc.ShardID,
		Receiver:  sourceShard,
		Content:   contentBytes,
	}

	return bc.sendMessageToShard(sourceShard, message)
}

// InitiateBalanceReconciliation starts balance reconciliation with another shard
func (bc *BankCommunication) InitiateBalanceReconciliation(targetShard int,
	periodStart, periodEnd int64, transactions []*LoanTransfer) error {

	// Calculate net balance
	netBalance := big.NewInt(0)
	for _, tx := range transactions {
		if tx.FromShard == bc.ShardID && tx.ToShard == targetShard {
			// We sent money to target shard
			netBalance.Sub(netBalance, tx.Amount)
		} else if tx.FromShard == targetShard && tx.ToShard == bc.ShardID {
			// We received money from target shard
			netBalance.Add(netBalance, tx.Amount)
		}
	}

	content := BankBalanceReconciliationContent{
		PeriodStart:      periodStart,
		PeriodEnd:        periodEnd,
		NetBalance:       netBalance,
		Transactions:     transactions,
		ReconciliationID: fmt.Sprintf("recon-%d-%d-%d", bc.ShardID, targetShard, time.Now().Unix()),
	}

	contentBytes, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("failed to marshal reconciliation: %v", err)
	}

	message := BankMessage{
		Type:      BankBalanceReconciliation,
		Timestamp: time.Now().Unix(),
		Sender:    bc.ShardID,
		Receiver:  targetShard,
		Content:   contentBytes,
	}

	return bc.sendMessageToShard(targetShard, message)
}

// TransferLoanOwnership transfers loan ownership to another shard
func (bc *BankCommunication) TransferLoanOwnership(loanID string, borrower string,
	amount *big.Int, interest *big.Int, targetShard int) error {

	content := BankLoanTransferContent{
		LoanID:       loanID,
		Borrower:     borrower,
		Amount:       new(big.Int).Set(amount),
		Interest:     new(big.Int).Set(interest),
		TransferTime: time.Now().Unix(),
	}

	contentBytes, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("failed to marshal loan transfer: %v", err)
	}

	message := BankMessage{
		Type:      BankLoanTransfer,
		Timestamp: time.Now().Unix(),
		Sender:    bc.ShardID,
		Receiver:  targetShard,
		Content:   contentBytes,
	}

	// Update pending loan status
	bc.lock.Lock()
	if pending, exists := bc.pendingLoans[loanID]; exists {
		pending.Status = PendingLoanTransferred
	}
	bc.lock.Unlock()

	return bc.sendMessageToShard(targetShard, message)
}

// HandleMessage processes incoming bank messages
func (bc *BankCommunication) HandleMessage(message BankMessage) error {
	switch message.Type {
	case BankLoanNotification:
		return bc.handleLoanNotification(message)
	case BankRepaymentConfirmation:
		return bc.handleRepaymentConfirmation(message)
	case BankBalanceReconciliation:
		return bc.handleBalanceReconciliation(message)
	case BankLoanTransfer:
		return bc.handleLoanTransfer(message)
	default:
		return fmt.Errorf("unknown message type: %s", message.Type)
	}
}

// handleLoanNotification processes incoming loan notifications
func (bc *BankCommunication) handleLoanNotification(message BankMessage) error {
	var content BankLoanNotificationContent
	if err := json.Unmarshal(message.Content, &content); err != nil {
		return fmt.Errorf("failed to unmarshal loan notification: %v", err)
	}

	log.Printf("Bank shard %d received loan notification: borrower=%s, loanID=%s, amount=%s, from shard %d",
		bc.ShardID, content.Borrower, content.LoanID, content.Amount.String(), message.Sender)

	// Record the loan in our bank manager
	if bc.BankManager != nil {
		// Create a loan record for tracking
		// Note: LoanRecord uses CreatedBlock and DueBlock instead of timestamps
		// For now, we'll just log the loan information

		// Add to pending loans for tracking
		bc.lock.Lock()
		bc.pendingLoans[content.LoanID] = &PendingLoan{
			LoanID:      content.LoanID,
			Borrower:    content.Borrower,
			Amount:      new(big.Int).Set(content.Amount),
			SourceShard: message.Sender,
			TargetShard: bc.ShardID,
			CreatedTime: time.Now().Unix(),
			Status:      PendingLoanNotified,
		}
		bc.lock.Unlock()

		// Store in bank manager (this would be a cross-shard loan tracking method)
		// bc.BankManager.AddCrossShardLoan(loanRecord)
		log.Printf("Recorded cross-shard loan %s from shard %d", content.LoanID, message.Sender)
	}

	return nil
}

// handleRepaymentConfirmation processes repayment confirmations
func (bc *BankCommunication) handleRepaymentConfirmation(message BankMessage) error {
	var content BankRepaymentConfirmationContent
	if err := json.Unmarshal(message.Content, &content); err != nil {
		return fmt.Errorf("failed to unmarshal repayment confirmation: %v", err)
	}

	log.Printf("Bank shard %d received repayment confirmation: loanID=%s, amount=%s, from shard %d",
		bc.ShardID, content.LoanID, content.Amount.String(), message.Sender)

	// Update pending loan status
	bc.lock.Lock()
	if pending, exists := bc.pendingLoans[content.LoanID]; exists {
		pending.Status = PendingLoanConfirmed
		log.Printf("Loan %s confirmed as repaid", content.LoanID)
	}
	bc.lock.Unlock()

	// Update bank manager
	if bc.BankManager != nil {
		// Mark loan as repaid in source shard's records
		// bc.BankManager.MarkCrossShardLoanRepaid(content.LoanID, content.Amount)
		log.Printf("Updated loan %s status to repaid", content.LoanID)
	}

	return nil
}

// handleBalanceReconciliation processes balance reconciliation
func (bc *BankCommunication) handleBalanceReconciliation(message BankMessage) error {
	var content BankBalanceReconciliationContent
	if err := json.Unmarshal(message.Content, &content); err != nil {
		return fmt.Errorf("failed to unmarshal balance reconciliation: %v", err)
	}

	log.Printf("Bank shard %d received balance reconciliation: period=%d-%d, net balance=%s, from shard %d",
		bc.ShardID, content.PeriodStart, content.PeriodEnd, content.NetBalance.String(), message.Sender)

	// Process reconciliation
	// In a full implementation, this would:
	// 1. Verify the transactions
	// 2. Calculate our own net balance
	// 3. Compare with received net balance
	// 4. Initiate settlement if needed

	log.Printf("Balance reconciliation %s received from shard %d",
		content.ReconciliationID, message.Sender)

	return nil
}

// handleLoanTransfer processes loan ownership transfers
func (bc *BankCommunication) handleLoanTransfer(message BankMessage) error {
	var content BankLoanTransferContent
	if err := json.Unmarshal(message.Content, &content); err != nil {
		return fmt.Errorf("failed to unmarshal loan transfer: %v", err)
	}

	log.Printf("Bank shard %d received loan transfer: loanID=%s, borrower=%s, amount=%s, from shard %d",
		bc.ShardID, content.LoanID, content.Borrower, content.Amount.String(), message.Sender)

	// Take ownership of the loan
	bc.lock.Lock()
	bc.pendingLoans[content.LoanID] = &PendingLoan{
		LoanID:      content.LoanID,
		Borrower:    content.Borrower,
		Amount:      new(big.Int).Set(content.Amount),
		SourceShard: message.Sender,
		TargetShard: bc.ShardID,
		CreatedTime: time.Now().Unix(),
		Status:      PendingLoanTransferred,
	}
	bc.lock.Unlock()

	log.Printf("Took ownership of loan %s from shard %d", content.LoanID, message.Sender)

	return nil
}

// sendMessageToShard sends a message to another shard's bank
func (bc *BankCommunication) sendMessageToShard(targetShard int, message BankMessage) error {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	// Create wrapper for transmission
	type BankMessageWrapper struct {
		Message []byte
		ShardID int
	}

	wrapper := BankMessageWrapper{
		Message: messageBytes,
		ShardID: targetShard,
	}

	wrapperBytes, err := json.Marshal(wrapper)
	if err != nil {
		return fmt.Errorf("failed to marshal wrapper: %v", err)
	}

	// Get target shard leader address
	targetLeader := ""
	shardKey := "S" + strconv.Itoa(targetShard)
	if nodes, ok := params.NodeTable[shardKey]; ok {
		if leader, ok := nodes["N0"]; ok {
			targetLeader = leader
		}
	}

	if targetLeader == "" {
		return fmt.Errorf("could not find leader for shard %d", targetShard)
	}

	// Record as pending loan before sending
	bc.lock.Lock()
	if message.Type == BankLoanNotification {
		var content BankLoanNotificationContent
		if err := json.Unmarshal(message.Content, &content); err == nil {
			bc.pendingLoans[content.LoanID] = &PendingLoan{
				LoanID:      content.LoanID,
				Borrower:    content.Borrower,
				Amount:      new(big.Int).Set(content.Amount),
				SourceShard: bc.ShardID,
				TargetShard: targetShard,
				CreatedTime: time.Now().Unix(),
				Status:      PendingLoanNotified,
			}
		}
	}
	bc.lock.Unlock()

	// Create command message manually (since we can't access pbft.jointMessage directly)
	cmd := "bankmessage"
	b := make([]byte, 12) // prefixCMDLength is 12
	for i, v := range []byte(cmd) {
		if i < 12 {
			b[i] = v
		}
	}
	messageToSend := append(b, wrapperBytes...)

	// Send using existing TCP infrastructure
	utils.TcpDial(messageToSend, targetLeader)

	log.Printf("Bank shard %d sent %s message to shard %d leader %s",
		bc.ShardID, message.Type, targetShard, targetLeader)

	return nil
}

// GetPendingLoans returns all pending loans
func (bc *BankCommunication) GetPendingLoans() map[string]*PendingLoan {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	// Return a copy
	result := make(map[string]*PendingLoan)
	for k, v := range bc.pendingLoans {
		result[k] = &PendingLoan{
			LoanID:      v.LoanID,
			Borrower:    v.Borrower,
			Amount:      new(big.Int).Set(v.Amount),
			SourceShard: v.SourceShard,
			TargetShard: v.TargetShard,
			CreatedTime: v.CreatedTime,
			Status:      v.Status,
		}
	}
	return result
}

// ClearCompletedLoans removes confirmed loans from pending list
func (bc *BankCommunication) ClearCompletedLoans() {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	for loanID, pending := range bc.pendingLoans {
		if pending.Status == PendingLoanConfirmed {
			delete(bc.pendingLoans, loanID)
		}
	}
}
