package entities

import (
	"time"
)

type AppStatesNames string

const (
	LastMonitoredTransactionIndex AppStatesNames = "lastMonitoredTransactionIndex"
)

type AppState struct {
	ID         int32          `json:"id" gorm:"column:id;type:int(11) NOT NULL AUTO_INCREMENT"`
	Name       AppStatesNames `json:"name" gorm:"column:name;type:varchar(100) COLLATE utf8_unicode_ci NOT NULL"`
	Value      string         `json:"value" gorm:"column:value;type:varchar(1000) COLLATE utf8_unicode_ci DEFAULT ''"`
	CreateTime time.Time      `json:"createTime" gorm:"column:createTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP;"`
	UpdateTime time.Time      `json:"updateTime" gorm:"column:updateTime;type:timestamp NOT NULL;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;"`
}
