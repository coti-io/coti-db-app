package entities

import (
	"db-sync/dto"
	"time"
)

type BaseTransactionsInputs struct {
	ID              int32     `gorm:"column:id"`
	TransactionId   int32     `gorm:"column:transactionId"`
	Hash            string    `gorm:"column:hash"`
	Name            string    `gorm:"column:name"`
	AddressHash     string    `gorm:"column:addressHash"`
	Amount          float64   `gorm:"column:amount"`
	InputCreateTime float64   `gorm:"column:inputCreateTime"`
	CreateTime      time.Time `gorm:"column:createTime;autoCreateTime"`
	UpdateTime      time.Time `gorm:"column:updateTime;autoUpdateTime:milli"`
}

func NewBaseTransactionsInputs(tx *dto.BaseTransactionsRes, transactionId int32) *BaseTransactionsInputs {
	instance := new(BaseTransactionsInputs)
	instance.TransactionId = transactionId
	instance.Hash = tx.Hash
	instance.Name = tx.Name
	instance.AddressHash = tx.AddressHash
	instance.Amount = tx.Amount
	instance.InputCreateTime = tx.CreateTime //time.Unix(int64(tx.CreateTime), 0)
	return instance
}
