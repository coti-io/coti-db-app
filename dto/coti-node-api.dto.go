package dto

import "database/sql"

type TransactionResponse struct {
	Hash                           string                `json: "hash"`
	Index                          int32                 `json: "index,omitempty"`
	Amount                         float64               `json: "amount"`
	AttachmentTime                 float64               `json: "attachmentTime"`
	IsValid                        sql.NullBool          `json: isValid`
	TransactionCreateTime          float64               `json: "transactionCreateTime"`
	LeftParentHash                 string                `json: "leftParentHash"`
	RightParentHash                string                `json: "rightParentHash"`
	SenderHash                     string                `json: "senderHash"`
	SenderTrustScore               float64               `json: "senderTrustScore"`
	TransactionConsensusUpdateTime float64               `json: "transactionConsensusUpdateTime"`
	TransactionDescription         string                `json: "transactionDescription"`
	TrustChainConsensus            bool                  `json: "trustChainConsensus"`
	TrustChainTrustScore           float64               `json: "trustChainTrustScore"`
	Type                           string                `json: "type"`
	BaseTransactionsRes            []BaseTransactionsRes `json:"baseTransactions"`
}

type BaseTransactionsRes struct {
	TransactionHash       string         `json: "transactionHash"`
	AddressHash           string         `json: "addressHash"`
	Amount                float64        `json: "amount"`
	FullnodeFeeCreateTime float64        `json: "fullnodeFeeCreateTime"`
	OriginalAmount        float64        `json: "originalAmount"`
	Hash                  string         `json: "hash"`
	Name                  string         `json: "name"`
	CreateTime            float64        `json: "createTime"`
	ReducedAmount         float64        `json: "reducedAmount"`
	ReceiverCreateTime    float64        `json: "receiverCreateTime"`
	ReceiverDescription   sql.NullString `json: "receiverDescription"`
}

type TransactionsIndexTip struct {
	Status    string `json: "status"`
	LastIndex int64  `json: "lastIndex"`
}

type TransactionByHashRequest struct {
	TransactionHashes []string `json: "transactionHashes"`
}
