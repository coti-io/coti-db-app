package entities

import (
	"database/sql"
	"time"

	"github.com/coti-io/coti-db-app/dto"
)

type InputBaseTransaction struct {
	ID              int32          `json:"id" gorm:"column:id;type:int(11) NOT NULL AUTO_INCREMENT"`
	TransactionId   int32          `json:"transactionId" gorm:"column:transactionId;type:int(11) NOT NULL;index:transactionId_INDEX"`
	Hash            string         `json:"hash" gorm:"column:hash;type:varchar(200) COLLATE utf8_unicode_ci NOT NULL"`
	Name            string         `json:"name" gorm:"column:name;type:varchar(45) COLLATE utf8_unicode_ci NOT NULL DEFAULT ''"`
	AddressHash     string         `json:"addressHash" gorm:"column:addressHash;type:varchar(200) COLLATE utf8_unicode_ci NOT NULL"`
	Amount          float64        `json:"amount" gorm:"column:amount;type:decimal(20,10) NOT NULL"`
	CurrencyHash    sql.NullString `json:"currencyHash" gorm:"column:currencyHash;type:varchar(200) COLLATE utf8_unicode_ci DEFAULT NULL"`
	InputCreateTime float64        `json:"inputCreateTime" gorm:"column:inputCreateTime;type:decimal(20,10) NOT NULL"`
	CreateTime      time.Time      `json:"createTime" gorm:"column:createTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP;"`
	UpdateTime      time.Time      `json:"updateTime" gorm:"column:updateTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;"`
}

func NewInputBaseTransaction(tx *dto.BaseTransactionsRes, transactionId int32) *InputBaseTransaction {
	instance := new(InputBaseTransaction)
	instance.TransactionId = transactionId
	instance.Hash = tx.Hash
	instance.Name = tx.Name
	instance.AddressHash = tx.AddressHash
	instance.Amount = tx.Amount
	instance.InputCreateTime = tx.CreateTime //time.Unix(int64(tx.CreateTime), 0)
	instance.CurrencyHash = tx.CurrencyHash
	return instance
}
