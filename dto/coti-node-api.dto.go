package dto

import (
	"github.com/shopspring/decimal"
)

type TransactionResponse struct {
	Hash                           string                `json:"hash"`
	Index                          *int32                `json:"index,omitempty"`
	Amount                         decimal.Decimal       `json:"amount"`
	AttachmentTime                 decimal.Decimal       `json:"attachmentTime"`
	IsValid                        *bool                 `json:"isValid"`
	CreateTime                     decimal.Decimal       `json:"createTime"`
	LeftParentHash                 *string               `json:"leftParentHash"`
	RightParentHash                *string               `json:"rightParentHash"`
	NodeHash                       *string               `json:"nodeHash"`
	SenderHash                     *string               `json:"senderHash"`
	SenderTrustScore               float64               `json:"senderTrustScore"`
	TransactionConsensusUpdateTime decimal.NullDecimal   `json:"transactionConsensusUpdateTime"`
	TransactionDescription         *string               `json:"transactionDescription"`
	TrustChainConsensus            bool                  `json:"trustChainConsensus"`
	TrustChainTrustScore           decimal.Decimal       `json:"trustChainTrustScore"`
	Type                           *string               `json:"type"`
	BaseTransactionsRes            []BaseTransactionsRes `json:"baseTransactions"`
}

type BaseTransactionsRes struct {
	TransactionHash                    *string                       `json:"transactionHash"`
	AddressHash                        string                        `json:"addressHash"`
	Amount                             decimal.Decimal               `json:"amount"`
	OriginalAmount                     decimal.NullDecimal           `json:"originalAmount"`
	Hash                               string                        `json:"hash"`
	Name                               string                        `json:"name"`
	CreateTime                         decimal.Decimal               `json:"createTime"`
	ReducedAmount                      decimal.Decimal               `json:"reducedAmount"`
	ReceiverDescription                *string                       `json:"receiverDescription"`
	CurrencyHash                       *string                       `json:"currencyHash"`
	OriginalCurrencyHash               *string                       `json:"originalCurrencyHash"`
	SignerHash                         *string                       `json:"signerHash"`
	TokenGenerationServiceResponseData TokenGenerationServiceDataRes `json:"tokenGenerationServiceResponseData"`
	TokenMintingServiceResponseData    TokenMintingServiceDataRes    `json:"tokenMintingServiceResponseData"`
	Event                              *string                       `json:"event"`
	HardFork                           *bool                         `json:"hardFork"`
}

type TokenGenerationServiceDataRes struct {
	OriginatorCurrencyData OriginatorCurrencyDataRes `json:"originatorCurrencyResponseData"`
	CurrencyTypeData       CurrencyTypeDataRes       `json:"currencyTypeResponseData"`
	FeeAmount              decimal.Decimal           `json:"feeAmount"`
}
type TokenMintingServiceDataRes struct {
	FeeAmount           decimal.Decimal `json:"feeAmount"`
	MintingCurrencyHash string          `json:"mintingCurrencyHash"`
	MintingAmount       decimal.Decimal `json:"mintingAmount"`
	ReceiverAddress     string          `json:"receiverAddress"`
	CreateTime          decimal.Decimal `json:"createTime"`
	SignerHash          string          `json:"signerHash"`
}

type OriginatorCurrencyDataRes struct {
	Name           *string         `json:"name"`
	Symbol         string          `json:"symbol"`
	Description    *string         `json:"description"`
	OriginatorHash *string         `json:"originatorHash"`
	TotalSupply    decimal.Decimal `json:"totalSupply"`
	Scale          int32           `json:"scale"`
}

type CurrencyTypeDataRes struct {
	CurrencyType           *string         `json:"currencyType"`
	CurrencyRateSourceType *string         `json:"currencyRateSourceType"`
	RateSource             *string         `json:"rateSource"`
	ProtectionModel        *string         `json:"protectionModel"`
	SignerHash             *string         `json:"SignerHash"`
	CreateTime             decimal.Decimal `json:"createTime"`
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
