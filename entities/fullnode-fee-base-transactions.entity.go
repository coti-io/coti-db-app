package entities

import (
	"time"

	"github.com/coti-io/coti-db-app/dto"
)

type FullnodeFeeBaseTransaction struct {
	ID                    int32     `json:"id" gorm:"column:id;type:int(11) NOT NULL AUTO_INCREMENT"`
	TransactionId         int32     `json:"transactionId" gorm:"column:transactionId;type:int(11) NOT NULL"`
	Hash                  string    `json:"hash" gorm:"column:hash;type:varchar(200) COLLATE utf8_unicode_ci NOT NULL"`
	AddressHash           string    `json:"addressHash" gorm:"column:addressHash;type:varchar(200) COLLATE utf8_unicode_ci NOT NULL"`
	Name                  string    `json:"name" gorm:"column:name;type:varchar(45) COLLATE utf8_unicode_ci NOT NULL DEFAULT ''"`
	Amount                float64   `json:"amount" gorm:"column:amount;type:decimal(20,10) NOT NULL"`
	FullnodeFeeCreateTime float64   `json:"fullnodeFeeCreateTime" gorm:"column:fullnodeFeeCreateTime;type:decimal(20,10) NOT NULL"`
	OriginalAmount        float64   `json:"originalAmount" gorm:"column:originalAmount;type:decimal(20,10)"`
	CreateTime            time.Time `json:"createTime" gorm:"column:createTime;type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP"`
	UpdateTime            time.Time `json:"updateTime" gorm:"column:updateTime;type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

func NewFullnodeFeeBaseTransaction(tx *dto.BaseTransactionsRes, transactionId int32) *FullnodeFeeBaseTransaction {
	instance := new(FullnodeFeeBaseTransaction)
	instance.TransactionId = transactionId
	instance.Hash = tx.Hash
	instance.Name = tx.Name
	instance.AddressHash = tx.AddressHash
	instance.Amount = tx.Amount
	instance.FullnodeFeeCreateTime = tx.CreateTime //time.Unix(int64(tx.CreateTime), 0)
	instance.OriginalAmount = tx.OriginalAmount
	return instance
}
