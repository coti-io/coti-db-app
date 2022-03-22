package entities

import (
	"github.com/shopspring/decimal"
	"time"
)

type TransactionAddress struct {
	ID             int32           `json:"id" gorm:"column:id;type:int(11) NOT NULL AUTO_INCREMENT"`
	TransactionId  int32           `json:"transactionId" gorm:"column:transactionId;type:int(11) NOT NULL"`
	AddressId      int32           `json:"addressId" gorm:"column:addressId;type:int(11) NOT NULL;index:addressId_INDEX"`
	AttachmentTime decimal.Decimal `json:"attachmentTime" gorm:"column:attachmentTime;type:decimal(20,6) NOT NULL;index:attachmentTime_INDEX"`
	CreateTime     time.Time       `json:"createTime" gorm:"column:createTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP;"`
	UpdateTime     time.Time       `json:"updateTime" gorm:"column:updateTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;"`
}

func NewTransactionAddress(addressId int32, attachmentTime decimal.Decimal, transactionId int32) *TransactionAddress {
	instance := new(TransactionAddress)
	instance.AttachmentTime = attachmentTime
	instance.AddressId = addressId
	instance.TransactionId = transactionId

	return instance
}

func (TransactionAddress) TableName() string {
	return "transaction_addresses"
}
