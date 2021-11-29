package entities

import (
	"database/sql"
	"db-sync/dto"
	"time"
)

type BaseTransaction struct {
	ID                             int32        `gorm:"column:id"`
	Hash                           string       `gorm:"column:hash"`
	Index                          int32        `gorm:"column:index"`
	Amount                         float64      `gorm:"column:amount"`
	AttachmentTime                 float64      `gorm:"column:attachmentTime"`
	IsValid                        sql.NullBool `gorm:"column:isValid"`
	TransactionCreateTime          float64      `gorm:"column:transactionCreateTime"`
	LeftParentHash                 string       `gorm:"column:leftParentHash"`
	RightParentHash                string       `gorm:"column:rightParentHash"`
	SenderHash                     string       `gorm:"column:senderHash"`
	SenderTrustScore               float64      `gorm:"column:senderTrustScore"`
	TransactionConsensusUpdateTime float64      `gorm:"column:transactionConsensusUpdateTime"`
	TransactionDescription         string       `gorm:"column:transactionDescription"`
	TrustChainConsensus            bool         `gorm:"column:trustChainConsensus"`
	TrustChainTrustScore           float64      `gorm:"column:trustChainTrustScore"`
	Type                           string       `gorm:"column:type"`
	CreateTime                     time.Time    `gorm:"column:createTime;autoCreateTime"`
	UpdateTime                     time.Time    `gorm:"column:updateTime;autoUpdateTime:milli"`
}

func NewBaseTransaction(tx *dto.TransactionResponse) *BaseTransaction {
	instance := new(BaseTransaction)
	instance.Hash = tx.Hash
	instance.Index = tx.Index
	instance.Amount = tx.Amount
	instance.IsValid = tx.IsValid
	instance.TransactionCreateTime = tx.TransactionCreateTime
	instance.LeftParentHash = tx.LeftParentHash
	instance.RightParentHash = tx.RightParentHash
	instance.SenderHash = tx.SenderHash
	instance.SenderTrustScore = tx.SenderTrustScore
	instance.TransactionConsensusUpdateTime = tx.TransactionConsensusUpdateTime
	instance.TransactionDescription = tx.TransactionDescription
	instance.TrustChainConsensus = tx.TrustChainConsensus
	instance.TrustChainTrustScore = tx.TrustChainTrustScore
	instance.Type = tx.Type
	return instance
}
