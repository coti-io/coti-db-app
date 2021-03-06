package entities

import (
	"github.com/shopspring/decimal"
	"time"

	"github.com/coti-io/coti-db-app/dto"
)

type TokenMintingServiceData struct {
	ID                    int32           `json:"id" gorm:"column:id;type:int(11) NOT NULL AUTO_INCREMENT"`
	BaseTransactionId     int32           `json:"baseTransactionId" gorm:"column:baseTransactionId;type:int(11) NOT NULL;index:baseTransactionId_INDEX"`
	MintingCurrencyHash   string          `json:"mintingCurrencyHash" gorm:"column:mintingCurrencyHash;type:varchar(200) COLLATE utf8_unicode_ci NOT NULL"`
	MintingAmount         decimal.Decimal `json:"mintingAmount" gorm:"column:mintingAmount;type:decimal(25,10) NOT NULL"`
	ServiceDataCreateTime decimal.Decimal `json:"serviceDataCreateTime" gorm:"column:serviceDataCreateTime;type:decimal(20,6) NOT NULL"`
	ReceiverAddress       string          `json:"receiverAddress" gorm:"column:receiverAddress;type:varchar(200) COLLATE utf8_unicode_ci NOT NULL"`
	FeeAmount             decimal.Decimal `json:"feeAmount" gorm:"column:feeAmount;type:decimal(25,10) NOT NULL"`
	SignerHash            string          `json:"signerHash" gorm:"column:signerHash;type:varchar(200) COLLATE utf8_unicode_ci NOT NULL"`
	CreateTime            time.Time       `json:"createTime" gorm:"column:createTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP;"`
	UpdateTime            time.Time       `json:"updateTime" gorm:"column:updateTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;"`
}

func (TokenMintingServiceData) TableName() string {
	return "token_minting_service_data"
}

func NewTokenMintingServiceData(sd *dto.TokenMintingServiceDataRes, baseTransactionId int32) *TokenMintingServiceData {
	instance := new(TokenMintingServiceData)
	instance.BaseTransactionId = baseTransactionId
	instance.MintingCurrencyHash = sd.MintingCurrencyHash
	instance.MintingAmount = sd.MintingAmount
	instance.ServiceDataCreateTime = sd.CreateTime
	instance.ReceiverAddress = sd.ReceiverAddress
	instance.SignerHash = sd.SignerHash
	instance.FeeAmount = sd.FeeAmount
	return instance
}
