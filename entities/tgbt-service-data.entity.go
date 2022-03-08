package entities

import (
	"github.com/shopspring/decimal"
	"time"

	"github.com/coti-io/coti-db-app/dto"
)

type TokenGenerationFeeServiceData struct {
	ID                int32           `json:"id" gorm:"column:id;type:int(11) NOT NULL AUTO_INCREMENT"`
	BaseTransactionId int32           `json:"baseTransactionId" gorm:"column:baseTransactionId;type:int(11) NOT NULL;index:transactionId_INDEX"`
	FeeAmount         decimal.Decimal `json:"feeAmount" gorm:"column:feeAmount;type:decimal(20,10) NOT NULL"`
	CreateTime        time.Time       `json:"createTime" gorm:"column:createTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP;"`
	UpdateTime        time.Time       `json:"updateTime" gorm:"column:updateTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;"`
}

func (TokenGenerationFeeServiceData) TableName() string {
	return "token_generation_fee_service_data"
}

func NewTokenGenerationFeeServiceData(sd *dto.TokenGenerationServiceDataRes, baseTransactionId int32) *TokenGenerationFeeServiceData {
	instance := new(TokenGenerationFeeServiceData)
	instance.BaseTransactionId = baseTransactionId
	instance.FeeAmount = sd.FeeAmount
	return instance
}
