package entities

import (
	"github.com/shopspring/decimal"
	"time"

	"github.com/coti-io/coti-db-app/dto"
)

type CurrencyTypeData struct {
	ID                         int32           `json:"id" gorm:"column:id;type:int(11) NOT NULL AUTO_INCREMENT"`
	ServiceDataId              int32           `json:"serviceDataId" gorm:"column:serviceDataId;type:int(11) NOT NULL;index:transactionId_INDEX"`
	CurrencyType               *string         `json:"currencyType" gorm:"column:currencyType;type:varchar(200) COLLATE utf8_unicode_ci DEFAULT NULL"`
	CurrencyRateSourceType     *string         `json:"currencyRateSourceType" gorm:"column:currencyRateSourceType;type:varchar(200) COLLATE utf8_unicode_ci DEFAULT NULL"`
	RateSource                 *string         `json:"rateSource" gorm:"column:rateSource;type:varchar(200) COLLATE utf8_unicode_ci DEFAULT NULL"`
	ProtectionModel            *string         `json:"protectionModel" gorm:"column:protectionModel;type:varchar(200) COLLATE utf8_unicode_ci DEFAULT NULL"`
	SignerHash                 *string         `json:"signerHash" gorm:"column:signerHash;type:varchar(200) COLLATE utf8_unicode_ci DEFAULT NULL"`
	CurrencyTypeDataCreateTime decimal.Decimal `json:"currencyTypeDataCreateTime" gorm:"column:currencyTypeDataCreateTime;type:decimal(20,6) NOT NULL"`
	CreateTime                 time.Time       `json:"createTime" gorm:"column:createTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP;"`
	UpdateTime                 time.Time       `json:"updateTime" gorm:"column:updateTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;"`
}

func (CurrencyTypeData) TableName() string {
	return "currency_type_data"
}

func NewCurrencyTypeData(currencyTypeData *dto.CurrencyTypeDataRes, serviceDataId int32) *CurrencyTypeData {
	instance := new(CurrencyTypeData)
	instance.ServiceDataId = serviceDataId
	instance.CurrencyType = currencyTypeData.CurrencyType
	instance.CurrencyRateSourceType = currencyTypeData.CurrencyRateSourceType
	instance.RateSource = currencyTypeData.RateSource
	instance.ProtectionModel = currencyTypeData.ProtectionModel
	instance.SignerHash = currencyTypeData.SignerHash
	instance.CurrencyTypeDataCreateTime = currencyTypeData.CreateTime
	return instance
}
