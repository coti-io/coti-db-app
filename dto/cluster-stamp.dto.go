package dto

import "github.com/shopspring/decimal"

type ClusterStampDataRow struct {
	Address    string
	Amount     decimal.Decimal
	CurrencyId int32
}
