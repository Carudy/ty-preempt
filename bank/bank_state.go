package bank

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"log"
	"math/big"
)

type BankState struct {
	Balance *big.Int
	ShardID int
}

func NewBankState(shardID int, initialBalance *big.Int) *BankState {
	return &BankState{
		Balance: new(big.Int).Set(initialBalance),
		ShardID: shardID,
	}
}

func (bs *BankState) Encode() []byte {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(bs)
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}

func DecodeBankState(to_decode []byte) *BankState {
	var state BankState
	decoder := gob.NewDecoder(bytes.NewReader(to_decode))
	err := decoder.Decode(&state)
	if err != nil {
		log.Panic(err)
	}
	return &state
}

func (bs *BankState) Hash() []byte {
	hash := sha256.Sum256(bs.Encode())
	return hash[:]
}

func (bs *BankState) CanLend(amount *big.Int) bool {
	return bs.Balance.Cmp(amount) >= 0
}

func (bs *BankState) Lend(amount *big.Int) bool {
	if bs.CanLend(amount) {
		bs.Balance.Sub(bs.Balance, amount)
		return true
	}
	return false
}

func (bs *BankState) ReceiveRepayment(amount *big.Int) {
	bs.Balance.Add(bs.Balance, amount)
}
