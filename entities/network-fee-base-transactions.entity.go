package entities

import (
	"database/sql"
	"time"

	"github.com/coti-io/coti-db-app/dto"
)

type NetworkFeeBaseTransaction struct {
	ID                   int32          `json:"id" gorm:"column:id;type:int(11) NOT NULL AUTO_INCREMENT"`
	TransactionId        int32          `json:"transactionId" gorm:"column:transactionId;type:int(11) NOT NULL;index:transactionId_INDEX"`
	Hash                 string         `json:"hash" gorm:"column:hash;type:varchar(200) COLLATE utf8_unicode_ci NOT NULL"`
	AddressHash          string         `json:"addressHash" gorm:"column:addressHash;type:varchar(200) COLLATE utf8_unicode_ci NOT NULL"`
	Amount               float64        `json:"amount" gorm:"column:amount;type:decimal(20,10) NOT NULL"`
	CurrencyHash         sql.NullString `json:"currencyHash" gorm:"column:currencyHash;type:varchar(200) COLLATE utf8_unicode_ci DEFAULT NULL"`
	Name                 string         `json:"name" gorm:"column:name;type:varchar(45) COLLATE utf8_unicode_ci NOT NULL DEFAULT ''"`
	NetworkFeeCreateTime float64        `json:"networkFeeCreateTime" gorm:"column:networkFeeCreateTime;type:decimal(20,10) NOT NULL"`
	OriginalAmount       float64        `json:"originalAmount" gorm:"column:originalAmount;type:decimal(20,10)"`
	OriginalCurrencyHash sql.NullString `json:"originalCurrencyHash" gorm:"column:originalCurrencyHash;type:varchar(200) COLLATE utf8_unicode_ci DEFAULT NULL"`
	ReducedAmount        float64        `json:"reducedAmount" gorm:"column:reducedAmount;type:decimal(20,10)"`
	CreateTime           time.Time      `json:"createTime" gorm:"column:createTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP;"`
	UpdateTime           time.Time      `json:"updateTime" gorm:"column:updateTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;"`
}

func NewNetworkFeeBaseTransaction(tx *dto.BaseTransactionsRes, transactionId int32) *NetworkFeeBaseTransaction {
	instance := new(NetworkFeeBaseTransaction)
	instance.TransactionId = transactionId
	instance.Hash = tx.Hash
	instance.Name = tx.Name
	instance.AddressHash = tx.AddressHash
	instance.Amount = tx.Amount
	instance.NetworkFeeCreateTime = tx.CreateTime //time.Unix(int64(tx.CreateTime), 0)
	instance.OriginalAmount = tx.OriginalAmount
	instance.ReducedAmount = tx.ReducedAmount
	instance.CurrencyHash = tx.CurrencyHash
	instance.OriginalCurrencyHash = tx.OriginalCurrencyHash
	return instance
}
