package service

import (
	"encoding/hex"
	"github.com/ebfe/keccak"
	"sync"
)

var currencyOnce sync.Once

type CurrencyService interface {
	// The exported functions
	getCurrencyHashBySymbol(symbol string) (error error, currencyHash string)
	normalizeCurrencyHash(currencyHash *string) string
}
type currencyService struct {
	// exported Fields
	nativeCurrencyHash string
}

var currencyServiceInstance *currencyService

func NewCurrencyService() CurrencyService {
	currencyOnce.Do(func() {
		currencyServiceInstance = &currencyService{
			nativeCurrencyHash: "e72d2137d5cfcc672ab743bddbdedb4e059ca9d3db3219f4eb623b01",
		}
	})
	return currencyServiceInstance
}

func (service *currencyService) getCurrencyHashBySymbol(symbol string) (error error, currencyHash string) {
	digest := keccak.New224()
	_, err := digest.Write([]byte(symbol))
	if err != nil {
		return err, ""
	}
	return nil, hex.EncodeToString(digest.Sum(nil))
}

func (service *currencyService) normalizeCurrencyHash(currencyHash *string) string {
	if currencyHash == nil {
		return service.nativeCurrencyHash
	}
	return *currencyHash
}
