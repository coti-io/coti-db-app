package entity

import "time"

type BaseTransactionsReceivers struct {
	id                  int32     `json: "id"`
	transactionId       int32     `json: "transactionId"`
	hash                string    `json: "hash"`
	addressHash         string    `json: "addressHash"`
	amount              float32   `json: "amount"`
	receiverCreateTime  time.Time `json: "receiverCreateTime"`
	originalAmount      float32   `json: "originalAmount"`
	receiverDescription string    `json: "receiverDescription"`
	createTime          time.Time `json: "createTime"`
	updateTime          time.Time `json: "updateTime"`
}
