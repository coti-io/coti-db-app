package entities

import (
	"coti-db-app/dto"
	"time"
)

type BaseTransactionFNF struct {
	ID                    int32     `gorm:"column:id"`
	TransactionId         int32     `gorm:"column:transactionId"`
	Hash                  string    `gorm:"column:hash"`
	AddressHash           string    `gorm:"column:addressHash"`
	Name                  string    `gorm:"column:name"`
	Amount                float64   `gorm:"column:amount"`
	FullnodeFeeCreateTime float64   `gorm:"column:fullnodeFeeCreateTime"`
	OriginalAmount        float64   `gorm:"column:originalAmount"`
	CreateTime            time.Time `gorm:"column:createTime;autoCreateTime"`
	UpdateTime            time.Time `gorm:"column:updateTime;autoUpdateTime:milli"`
}

func NewBaseTransactionFNF(tx *dto.BaseTransactionsRes, transactionId int32) *BaseTransactionFNF {
	instance := new(BaseTransactionFNF)
	instance.TransactionId = transactionId
	instance.Hash = tx.Hash
	instance.Name = tx.Name
	instance.AddressHash = tx.AddressHash
	instance.Amount = tx.Amount
	instance.FullnodeFeeCreateTime = tx.CreateTime //time.Unix(int64(tx.CreateTime), 0)
	instance.OriginalAmount = tx.OriginalAmount
	return instance
}

func (BaseTransactionFNF) TableName() string {
	return "base_transactions_fullnode_fees"
}
