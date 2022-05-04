package entities

import (
	"github.com/shopspring/decimal"
	"time"

	"github.com/coti-io/coti-db-app/dto"
)

type NetworkFeeBaseTransaction struct {
	ID                   int32               `json:"id" gorm:"column:id;type:int(11) NOT NULL AUTO_INCREMENT"`
	TransactionId        int32               `json:"transactionId" gorm:"column:transactionId;type:int(11) NOT NULL;index:transactionId_INDEX"`
	Hash                 string              `json:"hash" gorm:"column:hash;type:varchar(200) COLLATE utf8_unicode_ci NOT NULL"`
	AddressHash          string              `json:"addressHash" gorm:"column:addressHash;type:varchar(200) COLLATE utf8_unicode_ci NOT NULL"`
	Amount               decimal.Decimal     `json:"amount" gorm:"column:amount;type:decimal(25,10) NOT NULL"`
	CurrencyHash         *string             `json:"currencyHash" gorm:"column:currencyHash;type:varchar(200) COLLATE utf8_unicode_ci DEFAULT NULL"`
	Name                 string              `json:"name" gorm:"column:name;type:varchar(45) COLLATE utf8_unicode_ci NOT NULL DEFAULT ''"`
	NetworkFeeCreateTime decimal.Decimal     `json:"networkFeeCreateTime" gorm:"column:networkFeeCreateTime;type:decimal(20,6) NOT NULL"`
	OriginalAmount       decimal.NullDecimal `json:"originalAmount" gorm:"column:originalAmount;type:decimal(25,10)"`
	OriginalCurrencyHash *string             `json:"originalCurrencyHash" gorm:"column:originalCurrencyHash;type:varchar(200) COLLATE utf8_unicode_ci DEFAULT NULL"`
	ReducedAmount        decimal.Decimal     `json:"reducedAmount" gorm:"column:reducedAmount;type:decimal(25,10)"`
	CreateTime           time.Time           `json:"createTime" gorm:"column:createTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP;"`
	UpdateTime           time.Time           `json:"updateTime" gorm:"column:updateTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;"`
}

func NewNetworkFeeBaseTransaction(btx *dto.BaseTransactionsRes, transactionId int32) *NetworkFeeBaseTransaction {
	instance := new(NetworkFeeBaseTransaction)
	instance.TransactionId = transactionId
	instance.Hash = btx.Hash
	instance.Name = btx.Name
	instance.AddressHash = btx.AddressHash
	instance.Amount = btx.Amount
	instance.NetworkFeeCreateTime = btx.CreateTime //time.Unix(int64(tx.CreateTime), 0)
	instance.OriginalAmount = btx.OriginalAmount
	instance.ReducedAmount = btx.ReducedAmount
	instance.CurrencyHash = btx.CurrencyHash
	instance.OriginalCurrencyHash = btx.OriginalCurrencyHash
	return instance
}
