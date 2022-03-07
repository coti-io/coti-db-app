package service

import "strings"

type tokenBalance struct {
	CurrencyHash string
	AddressHash  string
}

func newTokenBalance(currencyHash string, addressHash string) *tokenBalance {
	instance := &tokenBalance{
		CurrencyHash: currencyHash,
		AddressHash:  addressHash,
	}
	return instance
}


func (service *tokenBalance) toString() string {
	return service.AddressHash + "_" + service.CurrencyHash
}
func newTokenBalanceFromString( tokenBalanceString string) *tokenBalance {
	parseResult := strings.Split(tokenBalanceString, "_")
	instance := &tokenBalance{
		CurrencyHash: parseResult[1],
		AddressHash:  parseResult[0],
	}
	return instance
}
