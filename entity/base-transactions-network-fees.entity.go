package entity

import "time"

type BaseTransactionsNF struct {
	id                    int32     `json: "id"`
	transactionId         int32     `json: "transactionId"`
	hash                  string    `json: "hash"`
	addressHash           string    `json: "addressHash"`
	amount                float32   `json: "amount"`
	fullnodeFeeCreateTime time.Time `json: "fullnodeFeeCreateTime"`
	originalAmount        float32   `json: "originalAmount"`
	reducedAmount         float32   `json: "reducedAmount"`
	createTime            time.Time `json: "createTime"`
	updateTime            time.Time `json: "updateTime"`
}
