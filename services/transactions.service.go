package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/shopspring/decimal"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	dbProvider "github.com/coti-io/coti-db-app/db-provider"
	"github.com/coti-io/coti-db-app/dto"
	"github.com/coti-io/coti-db-app/entities"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var transactionOnce sync.Once

type BaseTransactionName string

type SyncHistory struct {
	LastIndexMainNode   int64
	LastIndexBackupNode int64
	IsSynced            bool
}

func isResError(res *http.Response) bool {
	return res.StatusCode < 200 || res.StatusCode >= 300
}

type TransactionService interface {
	RunSync()
	GetLastIndex(fullnodeUrl string) <-chan dto.TransactionsLastIndexChanelResult
	GetLastIteration() int64
	GetFullnodeUrl() string
	GetBackupFullnodeUrl() string
	GetSyncHistory() SyncHistory
}
type transactionService struct {
	fullnodeUrl        string
	backupFullnodeUrl  string
	isSyncRunning      bool
	lastIterationIndex int64
	syncHistory        SyncHistory
	retries            uint8
	currentFullnodeUrl string
	serviceUpTime      time.Time
}

type UpdateBalanceRes struct {
	Id         int32
	Identifier string
}
type TxBuilder struct {
	Tx   dto.TransactionResponse
	DbTx *entities.Transaction
}

type TokenGenerationServiceDataBuilder struct {
	ServiceDataRes *dto.TokenGenerationServiceDataRes
	DbBaseTx       *entities.TokenGenerationFeeBaseTransaction
}

type TokenMintingServiceDataBuilder struct {
	ServiceDataRes *dto.TokenMintingServiceDataRes
	DbBaseTx       *entities.TokenMintingFeeBaseTransaction
	DbTx           *entities.Transaction
}

type TokenGenerationCurrencyBuilder struct {
	ServiceDataRes *dto.TokenGenerationServiceDataRes
	DbServiceData  *entities.TokenGenerationServiceData
	TxId           int32
}

type TransactionAddressBuilder struct {
	txId           int32
	addressHash    string
	attachmentTime decimal.Decimal
}

type TransactionCurrencyBuilder struct {
	txId           int32
	currencyHash   string
	attachmentTime decimal.Decimal
}

var instance *transactionService

// NewTransactionService we made this one a singleton because it has a state
func NewTransactionService() TransactionService {
	transactionOnce.Do(func() {

		instance = &transactionService{
			fullnodeUrl:        os.Getenv("FULLNODE_URL"),
			backupFullnodeUrl:  os.Getenv("FULLNODE_BACKUP_URL"),
			isSyncRunning:      false,
			lastIterationIndex: 0,
			syncHistory:        SyncHistory{LastIndexMainNode: 0, LastIndexBackupNode: 0, IsSynced: false},
			retries:            0,
			currentFullnodeUrl: os.Getenv("FULLNODE_URL"),
			serviceUpTime:      time.Now(),
		}
	})
	return instance
}

func (service *transactionService) GetFullnodeUrl() string {
	return service.fullnodeUrl
}
func (service *transactionService) GetBackupFullnodeUrl() string {
	return service.backupFullnodeUrl
}

func (service *transactionService) GetSyncHistory() SyncHistory {
	return service.syncHistory
}

// RunSync TODO: handle all errors by channels
func (service *transactionService) RunSync() {

	if service.isSyncRunning {
		return
	}
	service.isSyncRunning = true
	// run sync tasks
	go service.monitorSyncStatus()
	go service.syncNewTransactions(2)
	go service.monitorTransactions(2)
	go service.cleanUnindexedTransaction()
	go service.updateBalances()

}

func (service *transactionService) monitorSyncStatus() {
	iteration := 0
	for {
		dtStart := time.Now()
		fmt.Println("[monitorSyncStatus][iteration start] " + strconv.Itoa(iteration))
		iteration = iteration + 1
		err := service.monitorSyncStatusIteration()
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println("[monitorSyncStatus][iteration end] " + strconv.Itoa(iteration))
		dtEnd := time.Now()
		diff := dtEnd.Sub(dtStart)
		diffInSeconds := diff.Seconds()
		timeDurationToSleep := time.Duration(float64(10) - diffInSeconds)
		fmt.Println("[monitorSyncStatus][sleeping for] ", timeDurationToSleep)
		time.Sleep(timeDurationToSleep * time.Second)
		iteration += 1
	}
}

func (service *transactionService) monitorSyncStatusIteration() error {
	defer func() {
		if r := recover(); r != nil {
			log.Println("error in monitorSyncStatusIteration")
		}
	}()
	mainNodeCh, backupNodeCh := service.GetLastIndex(service.fullnodeUrl), service.GetLastIndex(service.backupFullnodeUrl)
	mainNodeRes, backupNodeRes := <-mainNodeCh, <-backupNodeCh
	isMainNodeError := mainNodeRes.Error != nil || mainNodeRes.Tran.Status == "error"
	isBackupNodeError := backupNodeRes.Error != nil || backupNodeRes.Tran.Status == "error"
	if isMainNodeError && isBackupNodeError {
		log.Println("Main fullnode and backup fullnode could not get last index")
		service.syncHistory.IsSynced = false

	} else if isMainNodeError {
		log.Println("Main fullnode could not get last index")
		service.syncHistory.IsSynced = false
	} else {
		if mainNodeRes.Tran.LastIndex >= backupNodeRes.Tran.LastIndex && service.lastIterationIndex >= mainNodeRes.Tran.LastIndex-100 {
			service.syncHistory.IsSynced = true

		} else if !isBackupNodeError {
			if mainNodeRes.Tran.LastIndex < service.syncHistory.LastIndexBackupNode &&
				float64(backupNodeRes.Tran.LastIndex-service.syncHistory.LastIndexBackupNode)*0.2 > float64(backupNodeRes.Tran.LastIndex-mainNodeRes.Tran.LastIndex) || service.lastIterationIndex < mainNodeRes.Tran.LastIndex-100 {
				service.syncHistory.IsSynced = false
			}
			service.syncHistory.LastIndexBackupNode = backupNodeRes.Tran.LastIndex
		} else {
			if service.lastIterationIndex >= mainNodeRes.Tran.LastIndex-100 {
				service.syncHistory.IsSynced = true
			}
		}
		service.syncHistory.LastIndexMainNode = mainNodeRes.Tran.LastIndex
	}
	return nil
}

func (service *transactionService) getAlternateNodeUrl(fullnodeUrl string) string {
	if fullnodeUrl == service.fullnodeUrl {
		return service.backupFullnodeUrl
	}
	return service.backupFullnodeUrl
}

func (service *transactionService) cleanUnindexedTransaction() {
	// when slice was less than 1000 once replace to the other method that gets un-indexed ones as well
	interval, err := strconv.ParseFloat(os.Getenv("CLEAN_UNINDEXED_TRANSACTIONS_INTERVAL_IN_SECONDS"), 64)
	if err != nil {
		panic(err.Error())
	}
	iteration := 0
	for {
		dtStart := time.Now()
		fmt.Println("[cleanUnindexedTransaction][iteration start] " + strconv.Itoa(iteration))
		iteration = iteration + 1
		err := service.cleanUnindexedTransactionIteration()
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println("[cleanUnindexedTransaction][iteration end] " + strconv.Itoa(iteration))
		dtEnd := time.Now()
		diff := dtEnd.Sub(dtStart)
		diffInSeconds := diff.Seconds()
		if diffInSeconds < interval && diffInSeconds > 0 {
			timeDurationToSleep := time.Duration(interval - diffInSeconds)
			fmt.Println("[cleanUnindexedTransaction][sleeping for] ", timeDurationToSleep)
			time.Sleep(timeDurationToSleep * time.Second)

		}
	}
}

func (service *transactionService) updateBalances() {
	// when slice was less than 1000 once replace to the other method that gets un-indexed ones as well
	interval, err := strconv.ParseFloat(os.Getenv("UPDATE_BALANCES_INTERVAL_IN_SECONDS"), 64)
	if err != nil {
		panic(err.Error())
	}
	iteration := 0
	for {
		dtStart := time.Now()
		fmt.Println("[updateBalances][iteration start] " + strconv.Itoa(iteration))
		iteration = iteration + 1
		err := service.updateBalancesIteration()
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println("[updateBalances][iteration end] " + strconv.Itoa(iteration))
		dtEnd := time.Now()
		diff := dtEnd.Sub(dtStart)
		diffInSeconds := diff.Seconds()
		if diffInSeconds < interval && diffInSeconds > 0 {
			timeDurationToSleep := time.Duration(interval - diffInSeconds)
			fmt.Println("[updateBalances][sleeping for] ", timeDurationToSleep)
			time.Sleep(timeDurationToSleep * time.Second)

		}
	}
}

func (service *transactionService) updateBalancesIteration() error {
	defer func() {
		if r := recover(); r != nil {
			log.Println("error in updateBalancesIteration")
		}
	}()
	err := dbProvider.DB.Transaction(func(dbTransaction *gorm.DB) (err error) {
		var appState entities.AppState
		err = dbTransaction.Clauses(clause.Locking{Strength: "UPDATE"}).Where("name = ?", entities.UpdateBalances).First(&appState).Error
		if err != nil {
			return err
		}

		var txs []entities.Transaction
		var ffbts []entities.FullnodeFeeBaseTransaction
		var nfbts []entities.NetworkFeeBaseTransaction
		var rbts []entities.ReceiverBaseTransaction
		var ibts []entities.InputBaseTransaction
		var tmbts []entities.TokenMintingFeeBaseTransaction
		var tgbts []entities.TokenGenerationFeeBaseTransaction
		var eibts []entities.EventInputBaseTransaction
		var tmbtServiceData []entities.TokenMintingServiceData

		// get all transaction with consensus and not processed
		err = dbTransaction.Where("`isProcessed` = 0 AND transactionConsensusUpdateTime IS NOT NULL AND type <> 'ZeroSpend'").Limit(3000).Find(&txs).Error
		if err != nil {
			return err
		}
		if len(txs) == 0 {
			fmt.Println("[updateBalancesIteration][no transactions to update balance was found]")
			return nil
		}
		txIdToAttachmentTime := make(map[int32]decimal.Decimal)
		for _, tx := range txs {
			txIdToAttachmentTime[tx.ID] = tx.AttachmentTime
		}
		var transactionIds []int32
		for i, v := range txs {
			txs[i].IsProcessed = true
			transactionIds = append(transactionIds, v.ID)
		}
		err = dbTransaction.Where(map[string]interface{}{"transactionId": transactionIds}).Find(&ffbts).Error
		if err != nil {
			return err
		}
		err = dbTransaction.Where(map[string]interface{}{"transactionId": transactionIds}).Find(&nfbts).Error
		if err != nil {
			return err
		}
		err = dbTransaction.Where(map[string]interface{}{"transactionId": transactionIds}).Find(&rbts).Error
		if err != nil {
			return err
		}
		err = dbTransaction.Where(map[string]interface{}{"transactionId": transactionIds}).Find(&ibts).Error
		if err != nil {
			return err
		}
		err = dbTransaction.Where(map[string]interface{}{"transactionId": transactionIds}).Find(&eibts).Error
		if err != nil {
			return err
		}
		err = dbTransaction.Where(map[string]interface{}{"transactionId": transactionIds}).Find(&tmbts).Error
		if err != nil {
			return err
		}
		err = dbTransaction.Where(map[string]interface{}{"transactionId": transactionIds}).Find(&tgbts).Error
		if err != nil {
			return err
		}
		err = dbTransaction.Save(&txs).Error
		if err != nil {
			return err
		}

		var tmbtIds []int32
		for _, v := range tmbts {
			tmbtIds = append(tmbtIds, v.ID)
		}
		err = dbTransaction.Where(map[string]interface{}{"baseTransactionId": tmbtIds}).Find(&tmbtServiceData).Error
		if err != nil {
			return err
		}

		serviceDataIdToTxId := make(map[int32]int32)
		for _, tmbtTx := range tmbts {
			for _, serviceDataTx := range tmbtServiceData {
				if tmbtTx.ID == serviceDataTx.BaseTransactionId {
					serviceDataIdToTxId[serviceDataTx.ID] = tmbtTx.TransactionId
				}
			}
		}

		uniqueHelperMap := make(map[string]bool)
		currencyHashUniqueArray := make([]string, 1)

		var currencyServiceInstance = NewCurrencyService()
		var addressBalanceDiffMap = make(map[string]decimal.Decimal)
		for _, baseTransaction := range tgbts {
			// calculate hash of currency
			currencyHash := currencyServiceInstance.NormalizeCurrencyHash(baseTransaction.CurrencyHash)
			addItemToUniqueArray(uniqueHelperMap, &currencyHashUniqueArray, currencyHash)
			btTokenBalance := newTokenBalance(currencyHash, baseTransaction.AddressHash)
			key := btTokenBalance.toString()
			addressBalanceDiffMap[key] = addressBalanceDiffMap[key].Add(baseTransaction.Amount)

		}

		for _, baseTransaction := range ffbts {
			currencyHash := currencyServiceInstance.NormalizeCurrencyHash(baseTransaction.CurrencyHash)
			addItemToUniqueArray(uniqueHelperMap, &currencyHashUniqueArray, currencyHash)
			btTokenBalance := newTokenBalance(currencyHash, baseTransaction.AddressHash)
			key := btTokenBalance.toString()
			addressBalanceDiffMap[key] = addressBalanceDiffMap[key].Add(baseTransaction.Amount)
		}
		for _, baseTransaction := range nfbts {
			currencyHash := currencyServiceInstance.NormalizeCurrencyHash(baseTransaction.CurrencyHash)
			addItemToUniqueArray(uniqueHelperMap, &currencyHashUniqueArray, currencyHash)
			btTokenBalance := newTokenBalance(currencyHash, baseTransaction.AddressHash)
			key := btTokenBalance.toString()
			addressBalanceDiffMap[key] = addressBalanceDiffMap[key].Add(baseTransaction.Amount)
		}
		for _, baseTransaction := range rbts {
			currencyHash := currencyServiceInstance.NormalizeCurrencyHash(baseTransaction.CurrencyHash)
			addItemToUniqueArray(uniqueHelperMap, &currencyHashUniqueArray, currencyHash)
			btTokenBalance := newTokenBalance(currencyHash, baseTransaction.AddressHash)
			key := btTokenBalance.toString()
			addressBalanceDiffMap[key] = addressBalanceDiffMap[key].Add(baseTransaction.Amount)
		}
		for _, baseTransaction := range ibts {
			currencyHash := currencyServiceInstance.NormalizeCurrencyHash(baseTransaction.CurrencyHash)
			addItemToUniqueArray(uniqueHelperMap, &currencyHashUniqueArray, currencyHash)
			btTokenBalance := newTokenBalance(currencyHash, baseTransaction.AddressHash)
			key := btTokenBalance.toString()
			addressBalanceDiffMap[key] = addressBalanceDiffMap[key].Add(baseTransaction.Amount)
		}
		for _, baseTransaction := range eibts {
			currencyHash := currencyServiceInstance.NormalizeCurrencyHash(baseTransaction.CurrencyHash)
			addItemToUniqueArray(uniqueHelperMap, &currencyHashUniqueArray, currencyHash)
			btTokenBalance := newTokenBalance(currencyHash, baseTransaction.AddressHash)
			key := btTokenBalance.toString()
			addressBalanceDiffMap[key] = addressBalanceDiffMap[key].Add(baseTransaction.Amount)
		}
		for _, baseTransaction := range tmbts {
			currencyHash := currencyServiceInstance.NormalizeCurrencyHash(baseTransaction.CurrencyHash)
			addItemToUniqueArray(uniqueHelperMap, &currencyHashUniqueArray, currencyHash)
			btTokenBalance := newTokenBalance(currencyHash, baseTransaction.AddressHash)
			key := btTokenBalance.toString()
			addressBalanceDiffMap[key] = addressBalanceDiffMap[key].Add(baseTransaction.Amount)
		}
		for _, serviceData := range tmbtServiceData {
			addItemToUniqueArray(uniqueHelperMap, &currencyHashUniqueArray, serviceData.MintingCurrencyHash)
			btTokenBalance := newTokenBalance(serviceData.MintingCurrencyHash, serviceData.ReceiverAddress)
			key := btTokenBalance.toString()
			fmt.Println(serviceData.MintingAmount.String())
			addressBalanceDiffMap[key] = addressBalanceDiffMap[key].Add(serviceData.MintingAmount)
		}

		err = updateBalances(dbTransaction, currencyHashUniqueArray, addressBalanceDiffMap)
		if err != nil {
			return err
		}

		return nil
	})
	return err
}

func appendTransactionCurrency(txId int32, attachmentTime decimal.Decimal, currencyHash *string, helperMapTransactionCurrencies map[string]bool, txCurrencyBuilders *[]*TransactionCurrencyBuilder, helperMapCurrencies map[string]bool) {
	var currencyServiceInstance = NewCurrencyService()
	var finalCurrencyHash string
	if currencyHash == nil {
		finalCurrencyHash = currencyServiceInstance.NormalizeCurrencyHash(currencyHash)
	} else {
		finalCurrencyHash = *currencyHash
	}
	uniqueTxIdCurrencyString := fmt.Sprintf("%d_%s", txId, finalCurrencyHash)
	if !helperMapTransactionCurrencies[uniqueTxIdCurrencyString] {
		*txCurrencyBuilders = append(*txCurrencyBuilders, &TransactionCurrencyBuilder{txId, finalCurrencyHash, attachmentTime})
		helperMapTransactionCurrencies[uniqueTxIdCurrencyString] = true
		helperMapCurrencies[finalCurrencyHash] = true
	}
}

func updateBalances(dbTransaction *gorm.DB, currencyHashUniqueArray []string, addressBalanceDiffMap map[string]decimal.Decimal) (err error) {
	// get all currency that have currency hash
	currenciesEntities := make([]entities.Currency, len(currencyHashUniqueArray))
	err = dbTransaction.Where(map[string]interface{}{"hash": currencyHashUniqueArray}).Find(&currenciesEntities).Error
	if err != nil {
		return err
	}
	var currencyHashToIdMap = make(map[string]int32)
	if len(currenciesEntities) > 0 {
		for _, c := range currenciesEntities {
			currencyHashToIdMap[c.Hash] = c.ID
		}
	}

	// get all address balances that exists
	addressHashes := make([]interface{}, 0, len(addressBalanceDiffMap))
	for key := range addressBalanceDiffMap {
		addressHashes = append(addressHashes, key)
	}

	// handle existing balance to update
	var updateBalanceResponseArray []UpdateBalanceRes

	err = dbTransaction.Model(&entities.AddressBalance{}).
		Select("address_balances.id, CONCAT(address_balances.addressHash, '_', currencies.hash) as identifier").
		Joins("INNER JOIN currencies on currencies.id = address_balances.currencyId").
		Where("CONCAT(address_balances.addressHash, '_', currencies.hash) IN (?"+strings.Repeat(",?", len(addressHashes)-1)+")", addressHashes...).
		Find(&updateBalanceResponseArray).Error
	if err != nil {
		return err
	}

	if len(updateBalanceResponseArray) > 0 {
		// get all ids to update
		var addressBalancesToUpdate []entities.AddressBalance
		var balanceIdsToUpdate []int32
		for _, v := range updateBalanceResponseArray {
			balanceIdsToUpdate = append(balanceIdsToUpdate, v.Id)
		}
		err = dbTransaction.Where(map[string]interface{}{"id": balanceIdsToUpdate}).Find(&addressBalancesToUpdate).Error
		if err != nil {
			return err
		}
		var mapIdToAddressBalance = make(map[int32]entities.AddressBalance)
		for _, ab := range addressBalancesToUpdate {
			mapIdToAddressBalance[ab.ID] = ab
		}
		var modifiedAddressBalancesToUpdate []entities.AddressBalance
		for _, adr := range updateBalanceResponseArray {
			// get balance diff by identifier
			balanceDiff := addressBalanceDiffMap[adr.Identifier]
			// get balanceToUpdate by id
			balanceToUpdate := mapIdToAddressBalance[adr.Id]
			// update balance
			balanceToUpdate.Amount = balanceToUpdate.Amount.Add(balanceDiff)
			modifiedAddressBalancesToUpdate = append(modifiedAddressBalancesToUpdate, balanceToUpdate)
			// remove from diff map
			delete(addressBalanceDiffMap, adr.Identifier)

		}
		if err := dbTransaction.Save(&modifiedAddressBalancesToUpdate).Error; err != nil {
			return err
		}

	}
	var addressBalancesToCreate []entities.AddressBalance
	for k, balanceDiff := range addressBalanceDiffMap {
		tb := newTokenBalanceFromString(k)
		currencyId := currencyHashToIdMap[tb.CurrencyHash]
		// create a new address balance
		addressBalance := entities.NewAddressBalance(tb.AddressHash, balanceDiff, currencyId)
		addressBalancesToCreate = append(addressBalancesToCreate, *addressBalance)
	}

	if len(addressBalancesToCreate) > 0 {
		if err := dbTransaction.Omit("CreateTime", "UpdateTime").Create(&addressBalancesToCreate).Error; err != nil {
			return err
		}
	}
	return nil
}

func updateAddressCounts(dbTransaction *gorm.DB, mapAddressTransactionCount map[string]int32) (err error) {
	// build array of addressHash unique array
	uniqueAddressHashArray := make([]string, 0, len(mapAddressTransactionCount))
	for k := range mapAddressTransactionCount {
		uniqueAddressHashArray = append(uniqueAddressHashArray, k)
	}
	// get all currency that have currency hash
	addressCountEntities := make([]entities.AddressTransactionCount, len(mapAddressTransactionCount))
	err = dbTransaction.Where(map[string]interface{}{"addressHash": uniqueAddressHashArray}).Find(&addressCountEntities).Error
	if err != nil {
		return err
	}
	var addressHashToAddressTransactionCountMap = make(map[string]entities.AddressTransactionCount)
	if len(addressCountEntities) > 0 {
		for _, ac := range addressCountEntities {
			addressHashToAddressTransactionCountMap[ac.AddressHash] = ac
		}
	}

	if len(addressCountEntities) > 0 {
		// get all ids to update
		var addressCountToUpdate []entities.AddressTransactionCount
		for _, ac := range addressCountEntities {
			ac.Count = ac.Count + mapAddressTransactionCount[ac.AddressHash]
			addressCountToUpdate = append(addressCountToUpdate, ac)
			// remove from diff map
			delete(mapAddressTransactionCount, ac.AddressHash)

		}
		if err := dbTransaction.Save(&addressCountToUpdate).Error; err != nil {
			return err
		}

	}
	var addressCountToCreate []entities.AddressTransactionCount
	for addressHash, count := range mapAddressTransactionCount {
		ac := entities.NewAddressTransactionCount(addressHash, count)
		addressCountToCreate = append(addressCountToCreate, *ac)
	}

	if len(addressCountToCreate) > 0 {
		if err := dbTransaction.Omit("CreateTime", "UpdateTime").Create(&addressCountToCreate).Error; err != nil {
			return err
		}
	}
	return nil
}

func (service *transactionService) cleanUnindexedTransactionIteration() error {
	defer func() {
		if r := recover(); r != nil {
			log.Println("[cleanUnindexedTransactionIteration][error]")
		}
	}()
	deleteTxDelayInHours, err := strconv.ParseFloat(os.Getenv("DELETE_TX_DELAY_IN_HOURS"), 64)

	if err != nil {
		return err
	}
	deleteTxPendingMinHours := os.Getenv("DELETE_TX_PENDING_MIN_HOURS")

	currTime := time.Now()
	diffTimeInHours := currTime.Sub(service.serviceUpTime).Hours()
	if diffTimeInHours < deleteTxDelayInHours {
		fmt.Println("[cleanUnindexedTransactionIteration][skip delete, time to start is not upon us]")
		return nil
	}
	err = dbProvider.DB.Transaction(func(dbTransaction *gorm.DB) error {
		var appState entities.AppState
		err = dbTransaction.Clauses(clause.Locking{Strength: "UPDATE"}).Where("name = ?", entities.DeleteUnindexedTransactions).First(&appState).Error
		if err != nil {
			return err
		}
		// get all indexed transaction or with status attached to dag from db
		var txs []entities.Transaction
		var txsAddresses []entities.TransactionAddress
		var txsCurrencies []entities.TransactionCurrency
		var ffbts []entities.FullnodeFeeBaseTransaction
		var nfbts []entities.NetworkFeeBaseTransaction
		var rbts []entities.ReceiverBaseTransaction
		var ibts []entities.InputBaseTransaction
		var eibts []entities.EventInputBaseTransaction
		var tgbt []entities.TokenGenerationFeeBaseTransaction
		var tmbt []entities.TokenMintingFeeBaseTransaction

		err = dbTransaction.Where("`index` IS NULL AND createTime < DATE_SUB(NOW(), INTERVAL ? HOUR)", deleteTxPendingMinHours).Find(&txs).Error
		if err != nil {
			return err
		}
		if len(txs) == 0 {
			fmt.Println("[cleanUnindexedTransactionIteration][no transactions to delete was found]")
			return nil
		}
		var newTxAppState entities.AppState
		err = dbTransaction.Clauses(clause.Locking{Strength: "UPDATE"}).Where("name = ?", entities.LastMonitoredTransactionIndex).First(&newTxAppState).Error
		if err != nil {
			return err
		}

		txs = make([]entities.Transaction, 0)
		if len(txs) == 0 {
			fmt.Println("[cleanUnindexedTransactionIteration][no transactions to delete was found]")
			return nil
		}

		// to operate addressTransactionCount
		helperMapAddressTransactionCount := make(map[string]bool)
		mapAddressTransactionCount := make(map[string]int32)

		var transactionIds []int32
		for _, v := range txs {
			transactionIds = append(transactionIds, v.ID)
		}
		err = dbTransaction.Where(map[string]interface{}{"transactionId": transactionIds}).Find(&tmbt).Error
		if err != nil {
			return err
		}
		if len(tmbt) > 0 {
			// get base transactions ids
			var tmbtBaseTransactionIds []int32
			var mapBtxIdToTxId = make(map[int32]int32)
			for _, v := range tmbt {
				mapBtxIdToTxId[v.ID] = v.TransactionId
				increaseCountIfUnique(helperMapAddressTransactionCount, mapAddressTransactionCount, fmt.Sprintf("%d_%s", v.TransactionId, v.AddressHash), v.AddressHash)
				tmbtBaseTransactionIds = append(tmbtBaseTransactionIds, v.ID)
			}

			// delete the service data
			var deletedTmbtsd []entities.TokenMintingServiceData
			err = dbTransaction.Where(map[string]interface{}{"baseTransactionId": tmbtBaseTransactionIds}).Delete(&deletedTmbtsd).Error
			if err != nil {
				return err
			}
			for _, v := range deletedTmbtsd {
				increaseCountIfUnique(helperMapAddressTransactionCount, mapAddressTransactionCount, fmt.Sprintf("%d_%s", mapBtxIdToTxId[v.ID], v.ReceiverAddress), v.ReceiverAddress)
			}
			err = dbTransaction.Where(map[string]interface{}{"transactionId": transactionIds}).Delete(&tmbt).Error
			if err != nil {
				return err
			}
		}

		err = dbTransaction.Where(map[string]interface{}{"transactionId": transactionIds}).Find(&tgbt).Error
		if err != nil {
			return err
		}
		if len(tgbt) > 0 {
			// get base transactions ids
			var tgbtBaseTransactionIds []int32
			for _, v := range tmbt {
				increaseCountIfUnique(helperMapAddressTransactionCount, mapAddressTransactionCount, fmt.Sprintf("%d_%s", v.TransactionId, v.AddressHash), v.AddressHash)
				tgbtBaseTransactionIds = append(tgbtBaseTransactionIds, v.ID)
			}
			// find the service data
			var tokenGenerationServiceData []entities.TokenGenerationServiceData
			err = dbTransaction.Where(map[string]interface{}{"baseTransactionId": tgbtBaseTransactionIds}).Find(&tokenGenerationServiceData).Error
			if err != nil {
				return err
			}
			// get base transactions ids
			var serviceDataIds []int32
			for _, v := range tmbt {

				serviceDataIds = append(serviceDataIds, v.ID)
			}
			var ocd []entities.OriginatorCurrencyData
			var ctd []entities.CurrencyTypeData
			err = dbTransaction.Where(map[string]interface{}{"transactionId": serviceDataIds}).Delete(&ocd).Error
			if err != nil {
				return err
			}
			err = dbTransaction.Where(map[string]interface{}{"transactionId": serviceDataIds}).Delete(&ctd).Error
			if err != nil {
				return err
			}
			err = dbTransaction.Where(map[string]interface{}{"baseTransactionId": tgbtBaseTransactionIds}).Delete(&tokenGenerationServiceData).Error
			if err != nil {
				return err
			}
			err = dbTransaction.Where(map[string]interface{}{"transactionId": transactionIds}).Delete(&tgbt).Error
			if err != nil {
				return err
			}
		}

		err = dbTransaction.Where(map[string]interface{}{"transactionId": transactionIds}).Delete(&ffbts).Error
		if err != nil {
			return err
		}
		for _, v := range ffbts {
			increaseCountIfUnique(helperMapAddressTransactionCount, mapAddressTransactionCount, fmt.Sprintf("%d_%s", v.TransactionId, v.AddressHash), v.AddressHash)
		}
		err = dbTransaction.Where(map[string]interface{}{"transactionId": transactionIds}).Delete(&nfbts).Error
		if err != nil {
			return err
		}
		for _, v := range nfbts {
			increaseCountIfUnique(helperMapAddressTransactionCount, mapAddressTransactionCount, fmt.Sprintf("%d_%s", v.TransactionId, v.AddressHash), v.AddressHash)
		}
		err = dbTransaction.Where(map[string]interface{}{"transactionId": transactionIds}).Delete(&rbts).Error
		if err != nil {
			return err
		}
		for _, v := range rbts {
			increaseCountIfUnique(helperMapAddressTransactionCount, mapAddressTransactionCount, fmt.Sprintf("%d_%s", v.TransactionId, v.AddressHash), v.AddressHash)
		}
		err = dbTransaction.Where(map[string]interface{}{"transactionId": transactionIds}).Delete(&ibts).Error
		if err != nil {
			return err
		}
		for _, v := range ibts {
			increaseCountIfUnique(helperMapAddressTransactionCount, mapAddressTransactionCount, fmt.Sprintf("%d_%s", v.TransactionId, v.AddressHash), v.AddressHash)
		}
		err = dbTransaction.Where(map[string]interface{}{"transactionId": transactionIds}).Delete(&eibts).Error
		if err != nil {
			return err
		}
		for _, v := range eibts {
			increaseCountIfUnique(helperMapAddressTransactionCount, mapAddressTransactionCount, fmt.Sprintf("%d_%s", v.TransactionId, v.AddressHash), v.AddressHash)
		}
		err = dbTransaction.Where(map[string]interface{}{"id": transactionIds}).Delete(&txs).Error
		err = dbTransaction.Where(map[string]interface{}{"transactionId": transactionIds}).Delete(&txsAddresses).Error
		err = dbTransaction.Where(map[string]interface{}{"transactionId": transactionIds}).Delete(&txsCurrencies).Error

		if err != nil {
			return err
		}
		// reverse the amount to decrease
		for k, v := range mapAddressTransactionCount {
			mapAddressTransactionCount[k] = -v
		}
		err = updateAddressCounts(dbTransaction, mapAddressTransactionCount)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (service *transactionService) syncNewTransactions(maxRetries uint8) {
	maxTransactionsInSync, err := strconv.ParseInt(os.Getenv("MAX_TRANSACTION_IN_SYNC_ITERATION"), 0, 64)
	if err != nil {
		panic(err.Error())
	}
	interval, err := strconv.ParseFloat(os.Getenv("SYNC_NEW_TRANSACTIONS_INTERVAL_IN_SECONDS"), 64)
	if err != nil {
		panic(err.Error())
	}
	var includeUnindexed = false

	// when slice was less than 1000 once replace to the other method that gets un-indexed ones as well
	iteration := 0
	for {
		iteration = iteration + 1
		dtStart := time.Now()
		fmt.Println("[syncNewTransactions][iteration start] " + strconv.Itoa(iteration))
		for {
			err := service.syncNewTransactionsIteration(maxTransactionsInSync, &includeUnindexed, service.currentFullnodeUrl)
			if err != nil {
				fmt.Println(err)
				if service.retries >= maxRetries {
					service.currentFullnodeUrl = service.getAlternateNodeUrl(service.currentFullnodeUrl)
					service.retries = 0
					break
				}
				service.retries = service.retries + 1
			} else {
				service.retries = 0
				break
			}

		}

		fmt.Println("[syncNewTransactions][iteration end] " + strconv.Itoa(iteration))
		dtEnd := time.Now()
		diff := dtEnd.Sub(dtStart)
		diffInSeconds := diff.Seconds()
		if diffInSeconds < interval && diffInSeconds > 0 && includeUnindexed {
			timeDurationToSleep := time.Duration(interval - diffInSeconds)
			fmt.Println("[syncNewTransactions][sleeping for] ", timeDurationToSleep)
			time.Sleep(timeDurationToSleep * time.Second)

		}
	}

}

func (service *transactionService) syncNewTransactionsIteration(maxTransactionsInSync int64, includeUnindexed *bool, fullnodeUrl string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
		}
	}()
	err = dbProvider.DB.Transaction(func(dbTransaction *gorm.DB) error {
		var appState entities.AppState
		err := dbTransaction.Clauses(clause.Locking{Strength: "UPDATE"}).Where("name = ?", entities.LastMonitoredTransactionIndex).First(&appState).Error
		if err != nil {
			return err
		}
		var lastMonitoredIndex int64
		if appState.Value == "" {
			lastMonitoredIndex = -1
		} else {
			lastMonitoredIndexInt, err := strconv.Atoi(appState.Value)
			if err != nil {
				return err
			}
			lastMonitoredIndex = int64(lastMonitoredIndexInt)
		}
		// get the tip
		lastIndexDtoChannel := service.GetLastIndex(fullnodeUrl)
		lastIndexObj := <-lastIndexDtoChannel
		if lastIndexObj.Error != nil {
			return lastIndexObj.Error
		}
		service.lastIterationIndex = lastIndexObj.Tran.LastIndex
		startingIndex := lastMonitoredIndex + 1

		// making sure we don't handle too much in one iteration
		var endingIndex int64
		if service.lastIterationIndex > startingIndex+maxTransactionsInSync {
			endingIndex = startingIndex + maxTransactionsInSync - 1
		} else {
			endingIndex = service.lastIterationIndex
			*includeUnindexed = true
		}
		var includeIndexed = true
		if startingIndex > endingIndex {
			includeIndexed = false
		}
		err, transactions := service.getTransactions(startingIndex, endingIndex, includeIndexed, *includeUnindexed, fullnodeUrl)
		if err != nil {
			return err
		}
		if len(transactions) > 0 {
			// get all the transactions hash
			var txHashArray []interface{}
			for _, tx := range transactions {
				txHashArray = append(txHashArray, tx.Hash)
			}
			// find records with a tx hash like the one we got and filter them from the array
			var dbTransactionsRes []entities.Transaction
			err = dbTransaction.Where("hash IN (?"+strings.Repeat(",?", len(txHashArray)-1)+")", txHashArray...).Find(&dbTransactionsRes).Error
			if err != nil {
				return err
			}
			var newTransactions []dto.TransactionResponse

			largestIndex := 0
			for _, tx := range transactions {
				exists := false
				for i, dbTx := range dbTransactionsRes {
					if dbTx.Hash == tx.Hash {
						exists = true
						if tx.TransactionConsensusUpdateTime != dbTx.TransactionConsensusUpdateTime {
							dbTransactionsRes[i].TransactionConsensusUpdateTime = tx.TransactionConsensusUpdateTime
						}
						if tx.TrustChainConsensus != dbTx.TrustChainConsensus {
							dbTransactionsRes[i].TrustChainConsensus = tx.TrustChainConsensus
						}
						if tx.Index != dbTx.Index {
							dbTransactionsRes[i].Index = tx.Index
						}
						if tx.TrustChainTrustScore != dbTx.TrustChainTrustScore {
							dbTransactionsRes[i].TrustChainTrustScore = tx.TrustChainTrustScore
						}
					}
				}
				if largestIndex < int(*tx.Index) {
					largestIndex = int(*tx.Index)
				}
				if !exists {
					newTransactions = append(newTransactions, tx)
				}
			}
			if len(dbTransactionsRes) > 0 {
				if err := dbTransaction.Save(&dbTransactionsRes).Error; err != nil {
					return err
				}
			}

			if len(newTransactions) > 0 {

				// prepare all the transactions to be saved
				var entitiesToBeSaved []*entities.Transaction
				txHashToTxBuilderMap := map[string]TxBuilder{}
				for _, tx := range newTransactions {
					dbTx := entities.NewTransaction(&tx)
					entitiesToBeSaved = append(entitiesToBeSaved, dbTx)
					txHashToTxBuilderMap[dbTx.Hash] = TxBuilder{tx, dbTx}
				}

				// save all of them
				if len(entitiesToBeSaved) > 0 {
					if err := dbTransaction.Omit("CreateTime", "UpdateTime").Create(&entitiesToBeSaved).Error; err != nil {
						return err
					}
				}

				if err := service.insertBaseTransactionsInputsOutputs(txHashToTxBuilderMap, dbTransaction); err != nil {
					return err
				}

			}

			if int64(largestIndex) > lastMonitoredIndex {
				appState.Value = strconv.Itoa(largestIndex)

				if err := dbTransaction.Omit("CreateTime", "UpdateTime").Save(&appState).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
	return err
}

func (service *transactionService) monitorTransactions(maxRetries uint8) {
	iteration := 0
	interval, err := strconv.ParseFloat(os.Getenv("MONITOR_TRANSACTION_INTERVAL_IN_SECONDS"), 64)
	if err != nil {
		panic(err.Error())
	}
	for {
		iteration++
		dtStart := time.Now()
		fmt.Println("[monitorTransactions][iteration start] " + strconv.Itoa(iteration))

		for {
			err := service.monitorTransactionIteration(service.currentFullnodeUrl)
			if err != nil {
				fmt.Println(err)

				// retry or try with replacement
				if service.retries >= maxRetries {
					service.currentFullnodeUrl = service.getAlternateNodeUrl(service.currentFullnodeUrl)
					service.retries = 0
					break
				}
			} else {
				service.retries = 0
				break
			}
		}
		fmt.Println("[monitorTransactions][iteration end] " + strconv.Itoa(iteration))
		dtEnd := time.Now()
		diff := dtEnd.Sub(dtStart)
		diffInSeconds := diff.Seconds()
		if diffInSeconds < interval && diffInSeconds > 0 {
			timeDurationToSleep := time.Duration(interval - diffInSeconds)
			fmt.Println("[monitorTransactions][sleeping for] ", timeDurationToSleep)
			time.Sleep(timeDurationToSleep * time.Second)

		}
	}
}

func (service *transactionService) monitorTransactionIteration(fullnodeUrl string) error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("[monitorTransactionIteration][error] ")
		}
	}()

	err := dbProvider.DB.Transaction(func(dbTransaction *gorm.DB) error {
		// get all indexed transaction or with status attached to dag from db
		var appState entities.AppState
		err := dbTransaction.Clauses(clause.Locking{Strength: "UPDATE"}).Where("name = ?", entities.MonitorTransaction).First(&appState).Error
		if err != nil {
			return err
		}
		var dbTransactions []entities.Transaction
		err = dbProvider.DB.Where("`index` IS NOT NULL AND transactionConsensusUpdateTime IS NULL").Find(&dbTransactions).Error
		if err != nil {
			return err
		}
		m := map[string]entities.Transaction{}
		var hashArray []string
		for _, tx := range dbTransactions {
			m[tx.Hash] = tx
			hashArray = append(hashArray, tx.Hash)
		}
		if hashArray != nil {
			// get the transactions from the node
			transactions, err := service.getTransactionsByHash(hashArray, fullnodeUrl)
			if err != nil {
				return err
			}
			// update the transactions
			var transactionToSave []*entities.Transaction
			for _, tx := range transactions {
				isChanged := false
				txToSave := m[tx.Hash]
				if tx.TransactionConsensusUpdateTime != txToSave.TransactionConsensusUpdateTime {
					isChanged = true
					txToSave.TransactionConsensusUpdateTime = tx.TransactionConsensusUpdateTime
				}
				if tx.TrustChainConsensus != txToSave.TrustChainConsensus {
					isChanged = true
					txToSave.TrustChainConsensus = tx.TrustChainConsensus
				}
				if tx.Index != txToSave.Index {
					isChanged = true
					txToSave.Index = tx.Index
				}
				if tx.TrustChainTrustScore != txToSave.TrustChainTrustScore && txToSave.TrustChainTrustScore.LessThan(tx.TrustChainTrustScore) {
					isChanged = true
					txToSave.TrustChainTrustScore = tx.TrustChainTrustScore
				}
				if isChanged {
					transactionToSave = append(transactionToSave, &txToSave)
				}
			}
			if len(transactionToSave) > 0 {
				err := dbProvider.DB.Omit("CreateTime", "UpdateTime").Save(&transactionToSave).Error
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
	return err
}

func (service *transactionService) getTransactions(startingIndex int64, endingIndex int64, includeIndexed bool, includeUnindexed bool, fullnodeUrl string) (err error, transactionsResponse []dto.TransactionResponse) {
	var data []dto.TransactionResponse

	if includeIndexed {
		log.Printf("[getTransactions][Getting transactions from index %d to index %d]\n", startingIndex, endingIndex)
		values := map[string]string{"startingIndex": strconv.FormatInt(startingIndex, 10), "endingIndex": strconv.FormatInt(endingIndex, 10), "extended": "true", "includeRuntimeTrustScore": "true"}
		jsonData, err := json.Marshal(values)

		if err != nil {
			return err, nil
		}

		res, err := http.Post(fullnodeUrl+"/transaction_batch", "application/json",
			bytes.NewBuffer(jsonData))

		if err != nil {
			return err, nil
		}

		if isResError(res) {
			return errors.New(res.Status), nil
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err, nil
		}
		err = json.Unmarshal(body, &data)
		if err != nil {
			return err, nil
		}
	}

	if includeUnindexed {
		log.Println("[getTransactions][Getting unindexed transactions]")
		res, err := http.Get(fullnodeUrl + "/transaction/none-indexed/batch")

		if err != nil {
			return err, nil
		}

		if isResError(res) {
			return errors.New(res.Status), nil
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err, nil
		}

		var unindexedData []dto.TransactionResponse
		err = json.Unmarshal(body, &unindexedData)
		if err != nil {
			return err, nil
		}
		data = append(data, unindexedData...)
	}
	return nil, data
}

func (service *transactionService) getTransactionsByHash(hashArray []string, fullnodeUrl string) (txs []dto.TransactionResponse, err error) {
	values := map[string]interface{}{"transactionHashes": hashArray, "includeRuntimeTrustScore": "true"}
	jsonData, err := json.Marshal(values)

	if err != nil {
		return nil, err
	}

	res, err := http.Post(fullnodeUrl+"/transaction/multiple", "application/json",
		bytes.NewBuffer(jsonData))

	if err != nil {
		return nil, err
	}

	if isResError(res) {
		return nil, errors.New(res.Status)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var data []dto.TransactionResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (service *transactionService) GetLastIndex(fullnodeUrl string) <-chan dto.TransactionsLastIndexChanelResult {

	r := make(chan dto.TransactionsLastIndexChanelResult)
	go func() {
		defer close(r)
		channelResult := dto.TransactionsLastIndexChanelResult{}

		res, err := http.Get(fullnodeUrl + "/transaction/lastIndex")

		if err != nil {
			channelResult.Error = err
			r <- channelResult
			return

		}

		if isResError(res) {
			channelResult.Error = errors.New(res.Status)
			r <- channelResult
			return
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			channelResult.Error = err
			r <- channelResult
			return
		}

		var data dto.TransactionsLastIndex
		err = json.Unmarshal(body, &data)
		if err != nil {
			channelResult.Error = err
			r <- channelResult
			return
		}
		channelResult.Tran = data
		r <- channelResult
	}()

	return r
}

func (service *transactionService) GetLastIteration() int64 {
	return service.lastIterationIndex
}

func (service *transactionService) insertBaseTransactionsInputsOutputs(txHashToTxBuilderMap map[string]TxBuilder, db *gorm.DB) error {

	var currencyServiceInstance = NewCurrencyService()
	var currencies []*entities.Currency
	var ibtsToBeSaved []*entities.InputBaseTransaction
	var rbtsToBeSaved []*entities.ReceiverBaseTransaction
	var ffbtsToBeSaved []*entities.FullnodeFeeBaseTransaction
	var nfbtsToBeSaved []*entities.NetworkFeeBaseTransaction
	var tgbtsToBeSaved []*entities.TokenGenerationFeeBaseTransaction
	var tmbtsToBeSaved []*entities.TokenMintingFeeBaseTransaction
	var eibtsToBeSaved []*entities.EventInputBaseTransaction
	var transactionAddressesToBeSaved []*entities.TransactionAddress
	var addressesToBeSaved []*entities.Address

	var tgCurrencyBuilders []*TokenGenerationCurrencyBuilder
	var tgbtServiceDataBuilders []*TokenGenerationServiceDataBuilder
	var tmbtServiceDataBuilders []*TokenMintingServiceDataBuilder
	var txAddressBuilders []*TransactionAddressBuilder
	var transactionCurrenciesToBeSaved []*entities.TransactionCurrency
	var txCurrencyBuilders []*TransactionCurrencyBuilder

	helperMapTransactionCurrencies := make(map[string]bool)
	helperMapCurrencies := make(map[string]bool)
	helperMapAddressTransactionCount := make(map[string]bool)
	helperMapTransactionAddresses := make(map[string]bool)
	helperMapAddresses := make(map[string]bool)
	mapAddressTransactionCount := make(map[string]int32)
	mapTgbtServiceDataIdToTxId := make(map[int32]int32)

	txIdToAttachmentTime := make(map[int32]decimal.Decimal)
	for _, tx := range txHashToTxBuilderMap {
		txIdToAttachmentTime[tx.DbTx.ID] = tx.DbTx.AttachmentTime
	}

	for _, value := range txHashToTxBuilderMap {
		txId := value.DbTx.ID
		attachmentTime := value.DbTx.AttachmentTime
		for _, baseTransaction := range value.Tx.BaseTransactionsRes {
			appendTransactionCurrency(txId, txIdToAttachmentTime[txId], baseTransaction.CurrencyHash, helperMapTransactionCurrencies, &txCurrencyBuilders, helperMapCurrencies)
			uniqueTxIdAddressString := fmt.Sprintf("%d_%s", txId, baseTransaction.AddressHash)
			if !helperMapTransactionAddresses[uniqueTxIdAddressString] {
				txAddressBuilders = append(txAddressBuilders, &TransactionAddressBuilder{txId, baseTransaction.AddressHash, attachmentTime})
				helperMapTransactionAddresses[uniqueTxIdAddressString] = true
			}
			if !helperMapAddresses[baseTransaction.AddressHash] {
				helperMapAddresses[baseTransaction.AddressHash] = true
				addressesToBeSaved = append(addressesToBeSaved, entities.NewAddress(baseTransaction.AddressHash))
			}
			increaseCountIfUnique(helperMapAddressTransactionCount, mapAddressTransactionCount, uniqueTxIdAddressString, baseTransaction.AddressHash)
			switch baseTransaction.Name {
			case "IBT":
				ibt := entities.NewInputBaseTransaction(&baseTransaction, txId)
				ibtsToBeSaved = append(ibtsToBeSaved, ibt)
			case "RBT":
				rbt := entities.NewReceiverBaseTransaction(&baseTransaction, txId)
				rbtsToBeSaved = append(rbtsToBeSaved, rbt)
			case "FFBT":
				ffbt := entities.NewFullnodeFeeBaseTransaction(&baseTransaction, txId)
				ffbtsToBeSaved = append(ffbtsToBeSaved, ffbt)
			case "NFBT":
				nfbt := entities.NewNetworkFeeBaseTransaction(&baseTransaction, txId)
				nfbtsToBeSaved = append(nfbtsToBeSaved, nfbt)
			case "TGBT":
				tgbt := entities.NewTokenGenerationFeeBaseTransaction(&baseTransaction, txId)
				tgbtsToBeSaved = append(tgbtsToBeSaved, tgbt)
				tgbtServiceDataBuilders = append(tgbtServiceDataBuilders, &TokenGenerationServiceDataBuilder{ServiceDataRes: &baseTransaction.TokenGenerationServiceData, DbBaseTx: tgbt})
				_, newCurrencyHash := currencyServiceInstance.GetCurrencyHashBySymbol(baseTransaction.TokenGenerationServiceData.OriginatorCurrencyData.Symbol)
				appendTransactionCurrency(txId, txIdToAttachmentTime[txId], &newCurrencyHash, helperMapTransactionCurrencies, &txCurrencyBuilders, helperMapCurrencies)
			case "TMBT":
				tmbt := entities.NewTokenMintingFeeBaseTransaction(&baseTransaction, txId)
				tmbtsToBeSaved = append(tmbtsToBeSaved, tmbt)
				tmbtServiceDataBuilders = append(tmbtServiceDataBuilders, &TokenMintingServiceDataBuilder{ServiceDataRes: &baseTransaction.TokenMintingServiceData, DbBaseTx: tmbt, DbTx: value.DbTx})
				newCurrencyHash := baseTransaction.TokenMintingServiceData.MintingCurrencyHash
				appendTransactionCurrency(txId, txIdToAttachmentTime[txId], &newCurrencyHash, helperMapTransactionCurrencies, &txCurrencyBuilders, helperMapCurrencies)
			case "EIBT":
				eibt := entities.NewEventInputBaseTransaction(&baseTransaction, txId)
				eibtsToBeSaved = append(eibtsToBeSaved, eibt)
			default:
				fmt.Println("Unknown base transaction name: ", baseTransaction.Name)
			}
		}

	}

	if len(ibtsToBeSaved) > 0 {
		if err := db.Omit("CreateTime", "UpdateTime").Create(&ibtsToBeSaved).Error; err != nil {
			log.Println(err)
			return err
		}
	}

	if len(rbtsToBeSaved) > 0 {
		if err := db.Omit("CreateTime", "UpdateTime").Create(&rbtsToBeSaved).Error; err != nil {
			log.Println(err)
			return err
		}
	}
	if len(ffbtsToBeSaved) > 0 {
		if err := db.Omit("CreateTime", "UpdateTime").Create(&ffbtsToBeSaved).Error; err != nil {
			log.Println(err)
			return err
		}
	}
	if len(nfbtsToBeSaved) > 0 {
		if err := db.Omit("CreateTime", "UpdateTime").Create(&nfbtsToBeSaved).Error; err != nil {
			log.Println(err)
			return err
		}
	}
	if len(eibtsToBeSaved) > 0 {
		if err := db.Omit("CreateTime", "UpdateTime").Create(&eibtsToBeSaved).Error; err != nil {
			log.Println(err)
			return err
		}
	}
	if len(tgbtsToBeSaved) > 0 {
		if err := db.Omit("CreateTime", "UpdateTime").Create(&tgbtsToBeSaved).Error; err != nil {
			log.Println(err)
			return err
		}
		var tgbtServiceDataToBeSaved []*entities.TokenGenerationServiceData
		for _, tgbtServiceDataBuilder := range tgbtServiceDataBuilders {
			dbServiceData := entities.NewTokenGenerationServiceData(tgbtServiceDataBuilder.ServiceDataRes, tgbtServiceDataBuilder.DbBaseTx.ID)
			tgbtServiceDataToBeSaved = append(tgbtServiceDataToBeSaved, dbServiceData)
			tgCurrencyBuilders = append(tgCurrencyBuilders, &TokenGenerationCurrencyBuilder{ServiceDataRes: tgbtServiceDataBuilder.ServiceDataRes, DbServiceData: dbServiceData, TxId: tgbtServiceDataBuilder.DbBaseTx.TransactionId})
		}
		if err := db.Omit("CreateTime", "UpdateTime").Create(&tgbtServiceDataToBeSaved).Error; err != nil {
			log.Println(err)
			return err
		}
		if len(tgCurrencyBuilders) > 0 {
			var originatorCurrencyDataToBeSaved []*entities.OriginatorCurrencyData
			var currencyTypeDataToBeSaved []*entities.CurrencyTypeData
			for _, currencyBuilder := range tgCurrencyBuilders {
				mapTgbtServiceDataIdToTxId[currencyBuilder.DbServiceData.ID] = currencyBuilder.TxId
				currencyTypeDataToBeSaved = append(currencyTypeDataToBeSaved, entities.NewCurrencyTypeData(&currencyBuilder.ServiceDataRes.CurrencyTypeData, currencyBuilder.DbServiceData.ID))
				originatorCurrencyDataToBeSaved = append(originatorCurrencyDataToBeSaved, entities.NewOriginatorCurrencyData(&currencyBuilder.ServiceDataRes.OriginatorCurrencyData, currencyBuilder.DbServiceData.ID))
			}
			if len(currencyTypeDataToBeSaved) > 0 {
				if err := db.Omit("CreateTime", "UpdateTime").Create(&currencyTypeDataToBeSaved).Error; err != nil {
					log.Println(err)
					return err
				}
			}

			if len(originatorCurrencyDataToBeSaved) > 0 {
				if err := db.Omit("CreateTime", "UpdateTime").Create(&originatorCurrencyDataToBeSaved).Error; err != nil {
					log.Println(err)
					return err
				}
				for _, oct := range originatorCurrencyDataToBeSaved {
					_, newCurrencyHash := currencyServiceInstance.GetCurrencyHashBySymbol(oct.Symbol)
					currency := entities.Currency{OriginatorCurrencyDataId: oct.ID, Hash: newCurrencyHash, TransactionId: mapTgbtServiceDataIdToTxId[oct.ServiceDataId]}
					currencies = append(currencies, &currency)
				}
			}
		}
	}
	if len(tmbtsToBeSaved) > 0 {
		if err := db.Omit("CreateTime", "UpdateTime").Create(&tmbtsToBeSaved).Error; err != nil {
			log.Println(err)
			return err
		}
		var tmbtServiceDataToBeSaved []*entities.TokenMintingServiceData
		for _, tmbtServiceDataBuilder := range tmbtServiceDataBuilders {
			dbServiceData := entities.NewTokenMintingServiceData(tmbtServiceDataBuilder.ServiceDataRes, tmbtServiceDataBuilder.DbBaseTx.ID)
			tmbtServiceDataToBeSaved = append(tmbtServiceDataToBeSaved, dbServiceData)
			uniqueTxIdAddressString := fmt.Sprintf("%d_%s", tmbtServiceDataBuilder.DbBaseTx.TransactionId, dbServiceData.ReceiverAddress)
			if !helperMapTransactionAddresses[uniqueTxIdAddressString] {
				txAddressBuilders = append(txAddressBuilders, &TransactionAddressBuilder{addressHash: dbServiceData.ReceiverAddress, txId: tmbtServiceDataBuilder.DbBaseTx.TransactionId, attachmentTime: tmbtServiceDataBuilder.DbTx.AttachmentTime})
				helperMapTransactionAddresses[uniqueTxIdAddressString] = true
			}
			if !helperMapAddresses[dbServiceData.ReceiverAddress] {
				helperMapAddresses[dbServiceData.ReceiverAddress] = true
				addressesToBeSaved = append(addressesToBeSaved, entities.NewAddress(dbServiceData.ReceiverAddress))
			}
			increaseCountIfUnique(helperMapAddressTransactionCount, mapAddressTransactionCount, uniqueTxIdAddressString, dbServiceData.ReceiverAddress)
		}
		if err := db.Omit("CreateTime", "UpdateTime").Create(&tmbtServiceDataToBeSaved).Error; err != nil {
			log.Println(err)
			return err
		}
	}
	err := updateAddressCounts(db, mapAddressTransactionCount)
	if err != nil {
		return err
	}
	mapAddressHashToAddressId := make(map[string]int32)
	if len(addressesToBeSaved) > 0 {
		var addressHashes []string
		for _, v := range addressesToBeSaved {
			addressHashes = append(addressHashes, v.AddressHash)
		}
		var existingAddresses []*entities.Address
		err := db.Where(map[string]interface{}{"addressHash": addressHashes}).Find(&existingAddresses).Error
		if err != nil {
			return err
		}
		var newAddresses []*entities.Address
		for _, addressToBeSaved := range addressesToBeSaved {
			created := false
			for _, existingAddress := range existingAddresses {
				if addressToBeSaved.AddressHash == existingAddress.AddressHash {
					created = true
					break
				}
			}
			if !created {
				newAddresses = append(newAddresses, addressToBeSaved)
			}
		}
		if len(newAddresses) > 0 {
			err = db.Omit("CreateTime", "UpdateTime").Create(&newAddresses).Error
			if err != nil {
				log.Println(err)
				return err
			}
		}

		for _, addr := range existingAddresses {
			mapAddressHashToAddressId[addr.AddressHash] = addr.ID
		}
		for _, addr := range newAddresses {
			mapAddressHashToAddressId[addr.AddressHash] = addr.ID
		}
	}

	if len(currencies) > 0 {
		if err := db.Omit("CreateTime", "UpdateTime").Create(&currencies).Error; err != nil {
			log.Println(err)
			return err
		}
	}

	if len(txAddressBuilders) > 0 {
		for _, txAddressBuilder := range txAddressBuilders {
			addressId := mapAddressHashToAddressId[txAddressBuilder.addressHash]
			transactionAddressesToBeSaved = append(transactionAddressesToBeSaved, entities.NewTransactionAddress(addressId, txAddressBuilder.attachmentTime, txAddressBuilder.txId))
		}
		if err := db.Omit("CreateTime", "UpdateTime").Create(&transactionAddressesToBeSaved).Error; err != nil {
			log.Println(err)
			return err
		}
	}

	if len(txCurrencyBuilders) > 0 {
		var currencies []entities.Currency
		var currencyHashArray []string
		currencyHashToIdMap := make(map[string]int32)
		for hash := range helperMapCurrencies {
			currencyHashArray = append(currencyHashArray, hash)
		}

		err = db.Where(map[string]interface{}{"hash": currencyHashArray}).Find(&currencies).Error
		if err != nil {
			return err
		}
		for _, currency := range currencies {
			currencyHashToIdMap[currency.Hash] = currency.ID
		}

		for _, txCurrencyBuilder := range txCurrencyBuilders {
			currencyId := currencyHashToIdMap[txCurrencyBuilder.currencyHash]
			transactionCurrenciesToBeSaved = append(transactionCurrenciesToBeSaved, entities.NewTransactionCurrency(currencyId, txCurrencyBuilder.attachmentTime, txCurrencyBuilder.txId))
		}

		if err := db.Omit("CreateTime", "UpdateTime").Create(&transactionCurrenciesToBeSaved).Error; err != nil {
			log.Println(err)
			return err
		}
	}

	return nil

}
