package entities

import (
	"coti-db-app/dto"
	"database/sql"
	"time"
)

type BaseTransactionsReceivers struct {
	ID                  int32          `gorm:"column:id"`
	TransactionId       int32          `gorm:"column:transactionId"`
	Hash                string         `gorm:"column:hash"`
	Name                string         `gorm:"column:name"`
	AddressHash         string         `gorm:"column:addressHash"`
	Amount              float64        `gorm:"column:amount"`
	ReceiverCreateTime  float64        `gorm:"column:receiverCreateTime"`
	OriginalAmount      float64        `gorm:"column:originalAmount"`
	ReceiverDescription sql.NullString `gorm:"column:receiverDescription"`
	CreateTime          time.Time      `gorm:"column:createTime;autoCreateTime"`
	UpdateTime          time.Time      `gorm:"column:updateTime;autoUpdateTime:milli"`
}

func NewBaseTransactionsReceivers(tx *dto.BaseTransactionsRes, transactionId int32) *BaseTransactionsReceivers {
	instance := new(BaseTransactionsReceivers)
	instance.TransactionId = transactionId
	instance.Hash = tx.Hash
	instance.Name = tx.Name
	instance.AddressHash = tx.AddressHash
	instance.Amount = tx.Amount
	instance.ReceiverCreateTime = tx.CreateTime // time.Unix(int64(tx.ReceiverCreateTime), 0)
	instance.OriginalAmount = tx.OriginalAmount
	instance.ReceiverDescription = tx.ReceiverDescription
	return instance
}
