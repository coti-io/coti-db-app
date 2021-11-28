package entity

import "time"

type BaseTransaction struct {
	id                             int32     `json: "id"`
	hash                           string    `json: "hash"`
	index                          int32     `json: "index"`
	amount                         float32   `json: "amount"`
	attachmentTime                 float32   `json: "attachmentTime"`
	isValid                        int8      `json: isValid`
	transactionCreateTime          float64   `json: "transactionCreateTime"`
	leftParentHash                 string    `json: "leftParentHash"`
	rightParentHash                string    `json: "rightParentHash"`
	senderHash                     string    `json: "senderHash"`
	senderTrustScore               float32   `json: "senderTrustScore"`
	transactionConsensusUpdateTime float64   `json: "transactionConsensusUpdateTime"`
	transactionDescription         string    `json: "transactionDescription"`
	trustChainConsensus            int8      `json: "trustChainConsensus"`
	trustChainTrustScore           float32   `json: "trustChainTrustScore"`
	txType                         string    `json: "type"`
	createTime                     time.Time `json: "createTime"`
	updateTime                     time.Time `json: "updateTime"`
}
