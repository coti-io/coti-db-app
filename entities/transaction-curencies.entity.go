package entities

import (
	"github.com/shopspring/decimal"
	"time"
)

type TransactionCurrency struct {
	ID             int32           `json:"id" gorm:"column:id;type:int(11) NOT NULL AUTO_INCREMENT"`
	TransactionId  int32           `json:"transactionId" gorm:"column:transactionId;type:int(11) NOT NULL"`
	CurrencyId     int32           `json:"currencyId" gorm:"column:currencyId;type:int(11) NOT NULL;"`
	AttachmentTime decimal.Decimal `json:"attachmentTime" gorm:"column:attachmentTime;type:decimal(20,6) NOT NULL;"`
	CreateTime     time.Time       `json:"createTime" gorm:"column:createTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP;"`
	UpdateTime     time.Time       `json:"updateTime" gorm:"column:updateTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;"`
}

func NewTransactionCurrency(currencyId int32, attachmentTime decimal.Decimal, transactionId int32) *TransactionCurrency {
	instance := new(TransactionCurrency)
	instance.AttachmentTime = attachmentTime
	instance.CurrencyId = currencyId
	instance.TransactionId = transactionId

	return instance
}

func (TransactionCurrency) TableName() string {
	return "transaction_currencies"
}