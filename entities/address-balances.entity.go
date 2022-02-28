package entities

import (
	"github.com/coti-io/coti-db-app/dto"
	"github.com/shopspring/decimal"
	"time"
)

type AddressBalance struct {
	ID          int32           `json:"id" gorm:"column:id;type:int(11) NOT NULL AUTO_INCREMENT"`
	CurrencyId  int32           `json:"currencyId" gorm:"column:currencyId;type:int(11) NOT NULL"`
	AddressHash string          `json:"addressHash" gorm:"column:addressHash;type:varchar(200) COLLATE utf8_unicode_ci NOT NULL;index:addressHash_INDEX"`
	Amount      decimal.Decimal `json:"amount" gorm:"column:amount;type:decimal(20,10) NOT NULL"`
	CreateTime  time.Time       `json:"createTime" gorm:"column:createTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP;"`
	UpdateTime  time.Time       `json:"updateTime" gorm:"column:updateTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;"`
}

func (AddressBalance) TableName() string {
	return "address_balances"
}

func NewAddressBalanceFromClusterStamp(clusterStampDataRow *dto.ClusterStampDataRow) *AddressBalance {
	instance := new(AddressBalance)
	instance.AddressHash = clusterStampDataRow.Address
	instance.Amount = clusterStampDataRow.Amount
	instance.CurrencyId = clusterStampDataRow.CurrencyId

	return instance
}

func NewAddressBalance(address string, amount decimal.Decimal, currencyId int32) *AddressBalance {
	instance := new(AddressBalance)
	instance.AddressHash = address
	instance.Amount = amount
	instance.CurrencyId = currencyId

	return instance
}
