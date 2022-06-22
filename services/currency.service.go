package service

import (
	"encoding/hex"
	"github.com/ebfe/keccak"
	"os"
	"strings"
	"sync"
)

var currencyOnce sync.Once

type CurrencyService interface {
	// The exported functions
	NormalizeCurrencyHash(currencyHash *string) string
	GetNativeCurrencyHash() string
	GetCurrencyHashBySymbol(symbol string) (err error, currencyHash string)
}
type currencyService struct {
	// exported Fields
	nativeCurrencyHash string
}

var currencyServiceInstance *currencyService

func NewCurrencyService() CurrencyService {
	currencyOnce.Do(func() {
		nativeSymbol := os.Getenv("NATIVE_SYMBOL")
		_, nativeCurrencyHashGenerated := getCurrencyHashBySymbol(nativeSymbol)
		currencyServiceInstance = &currencyService{
			nativeCurrencyHash: nativeCurrencyHashGenerated,
		}
	})
	return currencyServiceInstance
}

func (service *currencyService) NormalizeCurrencyHash(currencyHash *string) string {
	if currencyHash == nil {
		return service.nativeCurrencyHash
	}
	return *currencyHash
}

func (service *currencyService) GetNativeCurrencyHash() string {
	return service.nativeCurrencyHash
}

func (service *currencyService) GetCurrencyHashBySymbol(symbol string) (error error, currencyHash string) {
	return getCurrencyHashBySymbol(symbol)
}

func getCurrencyHashBySymbol(symbol string) (error error, currencyHash string) {
	digest := keccak.New224()
	_, err := digest.Write([]byte(strings.ToLower(symbol)))
	if err != nil {
		return err, ""
	}
	return nil, hex.EncodeToString(digest.Sum(nil))
}
