package entities

import (
	"github.com/shopspring/decimal"
	"time"
)

type TransactionAddress struct {
	ID             int32           `json:"id" gorm:"column:id;type:int(11) NOT NULL AUTO_INCREMENT"`
	TransactionId  int32           `json:"transactionId" gorm:"column:transactionId;type:int(11) NOT NULL"`
	AddressHash    string          `json:"addressHash" gorm:"column:addressHash;type:varchar(200) COLLATE utf8_unicode_ci NOT NULL;index:addressHash_INDEX"`
	AttachmentTime decimal.Decimal `json:"attachmentTime" gorm:"column:attachmentTime;type:decimal(20,6) NOT NULL;index:attachmentTime_INDEX"`
	CreateTime     time.Time       `json:"createTime" gorm:"column:createTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP;"`
	UpdateTime     time.Time       `json:"updateTime" gorm:"column:updateTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;"`
}

func NewTransactionAddress(addressHash string, attachmentTime decimal.Decimal, transactionId int32) *TransactionAddress {
	instance := new(TransactionAddress)
	instance.AttachmentTime = attachmentTime
	instance.AddressHash = addressHash
	instance.TransactionId = transactionId

	return instance
}

func (TransactionAddress) TableName() string {
	return "transaction_addresses"
}
