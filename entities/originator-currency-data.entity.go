package entities

import (
	"github.com/shopspring/decimal"
	"time"

	"github.com/coti-io/coti-db-app/dto"
)

type OriginatorCurrencyData struct {
	ID             int32           `json:"id" gorm:"column:id;type:int(11) NOT NULL AUTO_INCREMENT"`
	ServiceDataId  int32           `json:"serviceDataId" gorm:"column:serviceDataId;type:int(11) NOT NULL;index:serviceDataId_INDEX"`
	Name           *string         `json:"name" gorm:"column:name;type:varchar(200) COLLATE utf8_unicode_ci DEFAULT NULL"`
	Symbol         string          `json:"symbol" gorm:"column:symbol;type:varchar(200) COLLATE utf8_unicode_ci"`
	Description    *string         `json:"description" gorm:"column:description;type:varchar(500) COLLATE utf8_unicode_ci DEFAULT NULL"`
	OriginatorHash *string         `json:"originatorHash" gorm:"column:originatorHash;type:varchar(200) COLLATE utf8_unicode_ci DEFAULT NULL"`
	TotalSupply    decimal.Decimal `json:"totalSupply" gorm:"column:totalSupply;type:decimal(25,10) NOT NULL"`
	Scale          int32           `json:"scale" gorm:"column:scale;type:int(11) NOT NULL"`
	CreateTime     time.Time       `json:"createTime" gorm:"column:createTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP;"`
	UpdateTime     time.Time       `json:"updateTime" gorm:"column:updateTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;"`
}

func (OriginatorCurrencyData) TableName() string {
	return "originator_currency_data"
}

func NewOriginatorCurrencyData(originatorDataRes *dto.OriginatorCurrencyDataRes, serviceDataId int32) *OriginatorCurrencyData {
	instance := new(OriginatorCurrencyData)
	instance.ServiceDataId = serviceDataId
	instance.Name = originatorDataRes.Name
	instance.Symbol = originatorDataRes.Symbol
	instance.Description = originatorDataRes.Description
	instance.OriginatorHash = originatorDataRes.OriginatorHash
	instance.TotalSupply = originatorDataRes.TotalSupply
	instance.Scale = originatorDataRes.Scale
	return instance
}
