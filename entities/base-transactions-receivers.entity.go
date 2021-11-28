package entities

import (
	"db-sync/dto"
	"time"
)

type BaseTransactionsReceivers struct {
	ID                  int32     `gorm:"column:id"`
	TransactionId       int32     `gorm:"column:transactionId"`
	Hash                string    `gorm:"column:hash"`
	AddressHash         string    `gorm:"column:addressHash"`
	Amount              float64   `gorm:"column:amount"`
	ReceiverCreateTime  float64   `gorm:"column:receiverCreateTime"`
	OriginalAmount      float64   `gorm:"column:originalAmount"`
	ReceiverDescription string    `gorm:"column:receiverDescription"`
	CreateTime          time.Time `gorm:"column:createTime;autoCreateTime"`
	UpdateTime          time.Time `gorm:"column:updateTime;autoUpdateTime:milli"`
}

func NewBaseTransactionsReceivers(tx *dto.BaseTransactionsRes, transactionId int32) *BaseTransactionsReceivers {
	instance := new(BaseTransactionsReceivers)
	instance.TransactionId = transactionId
	instance.Hash = tx.Hash
	instance.AddressHash = tx.AddressHash
	instance.Amount = tx.Amount
	instance.ReceiverCreateTime = tx.CreateTime // time.Unix(int64(tx.ReceiverCreateTime), 0)
	instance.OriginalAmount = tx.OriginalAmount
	instance.ReceiverDescription = tx.ReceiverDescription
	return instance
}
