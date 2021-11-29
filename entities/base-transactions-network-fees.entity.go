package entities

import (
	"coti-db-app/dto"
	"time"
)

type BaseTransactionsNF struct {
	ID                    int32     `gorm:"column:id"`
	TransactionId         int32     `gorm:"column:transactionId"`
	Hash                  string    `gorm:"column:hash"`
	AddressHash           string    `gorm:"column:addressHash"`
	Amount                float64   `gorm:"column:amount"`
	Name                  string    `gorm:"column:name"`
	FullnodeFeeCreateTime float64   `gorm:"column:fullnodeFeeCreateTime"`
	OriginalAmount        float64   `gorm:"column:originalAmount"`
	ReducedAmount         float64   `gorm:"column:reducedAmount"`
	CreateTime            time.Time `gorm:"column:createTime;autoCreateTime"`
	UpdateTime            time.Time `gorm:"column:updateTime;autoUpdateTime:milli"`
}

func NewBaseTransactionNF(tx *dto.BaseTransactionsRes, transactionId int32) *BaseTransactionsNF {
	instance := new(BaseTransactionsNF)
	instance.TransactionId = transactionId
	instance.Hash = tx.Hash
	instance.Name = tx.Name
	instance.AddressHash = tx.AddressHash
	instance.Amount = tx.Amount
	instance.FullnodeFeeCreateTime = tx.CreateTime //time.Unix(int64(tx.CreateTime), 0)
	instance.OriginalAmount = tx.OriginalAmount
	instance.ReducedAmount = tx.ReducedAmount
	return instance
}

func (BaseTransactionsNF) TableName() string {
	return "base_transactions_network_fees"
}
