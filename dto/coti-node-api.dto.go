package dto

import (
	"github.com/shopspring/decimal"
)

type TransactionResponse struct {
	Hash                           string                `json:"hash"`
	Index                          int32                 `json:"index,omitempty"`
	Amount                         decimal.Decimal       `json:"amount"`
	AttachmentTime                 decimal.NullDecimal   `json:"attachmentTime"`
	IsValid                        *bool                 `json:"isValid"`
	CreateTime                     decimal.Decimal       `json:"createTime"`
	LeftParentHash                 *string               `json:"leftParentHash"`
	RightParentHash                *string               `json:"rightParentHash"`
	SenderHash                     *string               `json:"senderHash"`
	SenderTrustScore               float64               `json:"senderTrustScore"`
	TransactionConsensusUpdateTime decimal.NullDecimal   `json:"transactionConsensusUpdateTime"`
	TransactionDescription         *string               `json:"transactionDescription"`
	TrustChainConsensus            bool                  `json:"trustChainConsensus"`
	TrustChainTrustScore           decimal.NullDecimal   `json:"trustChainTrustScore"`
	Type                           *string               `json:"type"`
	BaseTransactionsRes            []BaseTransactionsRes `json:"baseTransactions"`
}

type BaseTransactionsRes struct {
	TransactionHash       string              `json:"transactionHash"`
	AddressHash           string              `json:"addressHash"`
	Amount                decimal.Decimal     `json:"amount"`
	FullnodeFeeCreateTime float64             `json:"fullnodeFeeCreateTime"`
	OriginalAmount        decimal.NullDecimal `json:"originalAmount"`
	Hash                  string              `json:"hash"`
	Name                  string              `json:"name"`
	CreateTime            decimal.Decimal     `json:"createTime"`
	ReducedAmount         decimal.Decimal     `json:"reducedAmount"`
	ReceiverDescription   *string             `json:"receiverDescription"`
	CurrencyHash          *string             `json:"currencyHash"`
	OriginalCurrencyHash  *string             `json:"originalCurrencyHash"`
}

type TransactionsLastIndex struct {
	Status    string `json:"status"`
	LastIndex int64  `json:"lastIndex"`
}

type TransactionsLastIndexChanelResult struct {
	Tran  TransactionsLastIndex
	Error error
}

type TransactionByHashRequest struct {
	TransactionHashes []string `json:"transactionHashes"`
}
