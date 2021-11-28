package entities

import (
	"time"
)

type AppStatesNames string

const (
	LastMonitoredTransactionIndex AppStatesNames = "lastMonitoredTransactionIndex"
)

type AppState struct {
	ID         int32          `gorm:"column:id"`
	Name       AppStatesNames `gorm:"column:name"`
	Value      string         `gorm:"column:value"`
	CreateTime time.Time      `gorm:"column:createTime;autoCreateTime"`
	UpdateTime time.Time      `gorm:"column:updateTime;autoUpdateTime:milli"`
}

func NewAppState(tx *AppState) *AppState {
	instance := new(AppState)
	instance.Name = tx.Name
	instance.Value = tx.Value
	return instance
}
