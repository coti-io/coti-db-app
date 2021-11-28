package entity

import "time"

type BaseTransactionsInputs struct {
	id              int32     `json: "id"`
	transactionId   int32     `json: "transactionId"`
	hash            string    `json: "hash"`
	addressHash     string    `json: "addressHash"`
	amount          float32   `json: "amount"`
	inputCreateTime time.Time `json: "inputCreateTime"`
	createTime      time.Time `json: "createTime"`
	updateTime      time.Time `json: "updateTime"`
}
