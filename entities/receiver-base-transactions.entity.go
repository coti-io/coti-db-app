package entities

import (
	"github.com/shopspring/decimal"
	"time"

	"github.com/coti-io/coti-db-app/dto"
)

type ReceiverBaseTransaction struct {
	ID                   int32               `json:"id" gorm:"column:id;type:int(11) NOT NULL AUTO_INCREMENT"`
	TransactionId        int32               `json:"transactionId" gorm:"column:transactionId;type:int(11) NOT NULL;index:transactionId_INDEX"`
	Hash                 string              `json:"hash" gorm:"column:hash;type:varchar(200) COLLATE utf8_unicode_ci NOT NULL"`
	Name                 string              `json:"name" gorm:"column:name;type:varchar(45) COLLATE utf8_unicode_ci NOT NULL DEFAULT ''"`
	AddressHash          string              `json:"addressHash" gorm:"column:addressHash;type:varchar(200) COLLATE utf8_unicode_ci NOT NULL"`
	Amount               decimal.Decimal     `json:"amount" gorm:"column:amount;type:decimal(20,10) NOT NULL"`
	CurrencyHash         *string             `json:"currencyHash" gorm:"column:currencyHash;type:varchar(200) COLLATE utf8_unicode_ci DEFAULT NULL"`
	ReceiverCreateTime   decimal.Decimal     `json:"receiverCreateTime" gorm:"column:receiverCreateTime;type:decimal(20,10) NOT NULL"`
	OriginalAmount       decimal.NullDecimal `json:"originalAmount" gorm:"column:originalAmount;type:decimal(20,10)"`
	OriginalCurrencyHash *string             `json:"originalCurrencyHash" gorm:"column:originalCurrencyHash;type:varchar(200) COLLATE utf8_unicode_ci DEFAULT NULL"`
	ReceiverDescription  *string             `json:"receiverDescription" gorm:"column:receiverDescription;type:varchar(200) COLLATE utf8_unicode_ci DEFAULT NULL"`
	CreateTime           time.Time           `json:"createTime" gorm:"column:createTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP;"`
	UpdateTime           time.Time           `json:"updateTime" gorm:"column:updateTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;"`
}

func NewReceiverBaseTransaction(tx *dto.BaseTransactionsRes, transactionId int32) *ReceiverBaseTransaction {
	instance := new(ReceiverBaseTransaction)
	instance.TransactionId = transactionId
	instance.Hash = tx.Hash
	instance.Name = tx.Name
	instance.AddressHash = tx.AddressHash
	instance.Amount = tx.Amount
	instance.ReceiverCreateTime = tx.CreateTime // time.Unix(int64(tx.ReceiverCreateTime), 0)
	instance.OriginalAmount = tx.OriginalAmount
	instance.ReceiverDescription = tx.ReceiverDescription
	instance.CurrencyHash = tx.CurrencyHash
	instance.OriginalCurrencyHash = tx.OriginalCurrencyHash
	return instance
}
