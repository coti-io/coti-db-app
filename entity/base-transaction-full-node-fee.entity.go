package entity

import "time"

type BaseTransactionFNF struct {
	id                    int32     `json: "id"`
	transactionId         int32     `json: "transactionId"`
	transactionHash       string    `json: "transactionHash"`
	addressHash           string    `json: "addressHash"`
	amount                float32   `json: "amount"`
	fullnodeFeeCreateTime time.Time `json: "fullnodeFeeCreateTime"`
	originalAmount        float32   `json: "originalAmount"`
	createTime            time.Time `json: "createTime"`
	updateTime            time.Time `json: "updateTime"`
}
