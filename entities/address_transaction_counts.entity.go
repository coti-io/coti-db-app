package entities

import (
	"time"
)

type AddressTransactionCount struct {
	ID          int32     `json:"id" gorm:"column:id;type:int(11) NOT NULL AUTO_INCREMENT"`
	Count       int32     `json:"count" gorm:"column:count;type:int(11) NOT NULL"`
	AddressHash string    `json:"addressHash" gorm:"column:addressHash;type:varchar(200) COLLATE utf8_unicode_ci NOT NULL;index:addressHash_INDEX"`
	CreateTime  time.Time `json:"createTime" gorm:"column:createTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP;"`
	UpdateTime  time.Time `json:"updateTime" gorm:"column:updateTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;"`
}

func NewAddressTransactionCount(addressHash string, count int32) *AddressTransactionCount {
	instance := new(AddressTransactionCount)
	instance.AddressHash = addressHash
	instance.Count = count
	return instance
}
