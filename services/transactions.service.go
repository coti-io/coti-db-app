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

var once sync.Once

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

type txBuilder struct {
	Tx   dto.TransactionResponse
	DbTx *entities.Transaction
}

var instance *transactionService

// NewTransactionService we made this one a singleton because it has a state
func NewTransactionService() TransactionService {
	once.Do(func() {

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
		if diffInSeconds < 10 && diffInSeconds > 0 {
			timeDurationToSleep := time.Duration(float64(10) - diffInSeconds)
			fmt.Println("[cleanUnindexedTransaction][sleeping for] ", timeDurationToSleep)
			time.Sleep(timeDurationToSleep * time.Second)

		}
	}
}

func (service *transactionService) updateBalances() {
	// when slice was less than 1000 once replace to the other method that gets un-indexed ones as well
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
		if diffInSeconds < 10 && diffInSeconds > 0 {
			timeDurationToSleep := time.Duration(float64(10) - diffInSeconds)
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
		// get all transaction with consensus and not processed
		// get all indexed transaction or with status attached to dag from db
		var txs []entities.Transaction
		var ffbts []entities.FullnodeFeeBaseTransaction
		var nfbts []entities.NetworkFeeBaseTransaction
		var rbts []entities.ReceiverBaseTransaction
		var ibts []entities.InputBaseTransaction
		err = dbTransaction.Where("`isProcessed` = 0 AND transactionConsensusUpdateTime > 0 AND type <> 'ZeroSpend'").Limit(3000).Find(&txs).Error
		if err != nil {
			return err
		}
		if len(txs) == 0 {
			fmt.Println("[updateBalancesIteration][no transactions to delete was found]")
			return nil
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
		err = dbTransaction.Save(&txs).Error
		if err != nil {
			return err
		}
		var addressBalanceDiffMap = make(map[string]decimal.Decimal)
		for _, tx := range ffbts {
			addressBalanceDiffMap[tx.AddressHash] = addressBalanceDiffMap[tx.AddressHash].Add(tx.Amount)
		}
		for _, tx := range nfbts {
			addressBalanceDiffMap[tx.AddressHash] = addressBalanceDiffMap[tx.AddressHash].Add(tx.Amount)
		}
		for _, tx := range rbts {
			addressBalanceDiffMap[tx.AddressHash] = addressBalanceDiffMap[tx.AddressHash].Add(tx.Amount)
		}
		for _, tx := range ibts {
			addressBalanceDiffMap[tx.AddressHash] = addressBalanceDiffMap[tx.AddressHash].Add(tx.Amount)
		}
		nativeCurrencyHash := "e72d2137d5cfcc672ab743bddbdedb4e059ca9d3db3219f4eb623b01"
		nativeCurrency := entities.Currency{}
		nativeCurrencyError := dbTransaction.Where("hash = ?", nativeCurrencyHash).First(&nativeCurrency).Error
		if nativeCurrencyError != nil {
			return nativeCurrencyError
		}
		// get all address balances that exists
		addressHashes := make([]interface{}, 0, len(addressBalanceDiffMap))
		for key := range addressBalanceDiffMap {
			addressHashes = append(addressHashes, key)
		}
		var dbAddressBalanceToCreate []entities.AddressBalance
		var dbAddressBalanceRes []entities.AddressBalance
		err = dbTransaction.Where("addressHash IN (?"+strings.Repeat(",?", len(addressHashes)-1)+")", addressHashes...).Find(&dbAddressBalanceRes).Error
		if err != nil {
			return err
		}
		// can be improved with map of
		// update balance if exists
		for addressHash, balanceDiff := range addressBalanceDiffMap {
			exists := false
			for i, addressBalance := range dbAddressBalanceRes {
				if addressBalance.AddressHash == addressHash {
					exists = true
					oldBalance := addressBalance.Amount
					dbAddressBalanceRes[i].Amount = oldBalance.Add(balanceDiff)
				}
			}
			// create record if not exists
			if !exists {
				// create a new address balance
				dbAddressBalanceToCreate = append(dbAddressBalanceToCreate, *entities.NewAddressBalance(addressHash, balanceDiff, nativeCurrency.ID))

			}
		}
		if len(dbAddressBalanceToCreate) > 0 {
			if err := dbTransaction.Omit("CreateTime", "UpdateTime").Create(&dbAddressBalanceToCreate).Error; err != nil {
				return err
			}
		}

		if len(dbAddressBalanceRes) > 0 {
			if err := dbTransaction.Save(&dbAddressBalanceRes).Error; err != nil {
				return err
			}
		}

		return nil
	})
	return err
}

func (service *transactionService) cleanUnindexedTransactionIteration() error {
	defer func() {
		if r := recover(); r != nil {
			log.Println("error in cleanUnindexedTransactionIteration")
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
		var ffbts []entities.FullnodeFeeBaseTransaction
		var nfbts []entities.NetworkFeeBaseTransaction
		var rbts []entities.ReceiverBaseTransaction
		var ibts []entities.InputBaseTransaction
		err = dbTransaction.Where("`index` = 0 AND transactionConsensusUpdateTime = 0 AND createTime < DATE_SUB(NOW(), INTERVAL ? HOUR)", deleteTxPendingMinHours).Find(&txs).Error
		if err != nil {
			return err
		}
		if len(txs) == 0 {
			fmt.Println("[cleanUnindexedTransactionIteration][no transactions to delete was found]")
			return nil
		}
		var transactionIds []int32
		for _, v := range txs {
			transactionIds = append(transactionIds, v.ID)
		}
		err = dbTransaction.Where(map[string]interface{}{"transactionId": transactionIds}).Delete(&ffbts).Error
		if err != nil {
			return err
		}
		err = dbTransaction.Where(map[string]interface{}{"transactionId": transactionIds}).Delete(&nfbts).Error
		if err != nil {
			return err
		}
		err = dbTransaction.Where(map[string]interface{}{"transactionId": transactionIds}).Delete(&rbts).Error
		if err != nil {
			return err
		}
		err = dbTransaction.Where(map[string]interface{}{"transactionId": transactionIds}).Delete(&ibts).Error
		if err != nil {
			return err
		}
		err = dbTransaction.Where(map[string]interface{}{"id": transactionIds}).Delete(&txs).Error
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
	var maxTransactionsInSync int64 = 3000 // in the future replace by 1000 and export to config
	var includeUnindexed = false

	// when slice was less than 1000 once replace to the other method that gets un-indexed ones as well
	iteration := 0
	for {
		iteration = iteration + 1
		dtStart := time.Now()
		fmt.Println("[syncNewTransactions][iteration start] " + strconv.Itoa(iteration))
		for {
			err := service.syncTransactionsIteration(maxTransactionsInSync, &includeUnindexed, service.currentFullnodeUrl)
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
		if diffInSeconds < 10 && diffInSeconds > 0 && includeUnindexed {
			timeDurationToSleep := time.Duration(float64(10) - diffInSeconds)
			fmt.Println("[syncNewTransactions][sleeping for] ", timeDurationToSleep)
			time.Sleep(timeDurationToSleep * time.Second)

		}
	}

}

func (service *transactionService) syncTransactionsIteration(maxTransactionsInSync int64, includeUnindexed *bool, fullnodeUrl string) (err error) {
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
		transactions := service.getTransactions(startingIndex, endingIndex, includeIndexed, *includeUnindexed, fullnodeUrl)
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
				for _, dbTx := range dbTransactionsRes {
					if dbTx.Hash == tx.Hash && dbTx.Index == 0 {
						exists = true
						if tx.TransactionConsensusUpdateTime != dbTx.TransactionConsensusUpdateTime {
							dbTx.TransactionConsensusUpdateTime = tx.TransactionConsensusUpdateTime
						}
						if tx.TrustChainConsensus != dbTx.TrustChainConsensus {
							dbTx.TrustChainConsensus = tx.TrustChainConsensus
						}
						if tx.Index != dbTx.Index {
							dbTx.Index = tx.Index
						}
					}
				}
				if largestIndex < int(tx.Index) {
					largestIndex = int(tx.Index)
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
				var baseTransactionsToBeSaved []*entities.Transaction
				m := map[string]txBuilder{}
				for _, tx := range newTransactions {
					dbTx := entities.NewTransaction(&tx)
					baseTransactionsToBeSaved = append(baseTransactionsToBeSaved, dbTx)
					m[dbTx.Hash] = txBuilder{tx, dbTx}
				}

				// save all of them
				if len(baseTransactionsToBeSaved) > 0 {
					if err := dbTransaction.Omit("CreateTime", "UpdateTime").Create(&baseTransactionsToBeSaved).Error; err != nil {
						return err
					}
				}

				if err := service.insertBaseTransactionsInputsOutputs(m, dbTransaction); err != nil {
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
	for {
		iteration = iteration + 1
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
		if diffInSeconds < 5 && diffInSeconds > 0 {
			timeDurationToSleep := time.Duration(float64(5) - diffInSeconds)
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
		err = dbProvider.DB.Where("`index` > 0 AND transactionConsensusUpdateTime = 0").Find(&dbTransactions).Error
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

func (service *transactionService) getTransactions(startingIndex int64, endingIndex int64, includeIndexed bool, includeUnindexed bool, fullnodeUrl string) []dto.TransactionResponse {
	var data []dto.TransactionResponse

	if includeIndexed {
		log.Printf("[getTransactions][Getting transactions from index %d to index %d]\n", startingIndex, endingIndex)
		values := map[string]string{"startingIndex": strconv.FormatInt(startingIndex, 10), "endingIndex": strconv.FormatInt(endingIndex, 10)}
		jsonData, err := json.Marshal(values)

		if err != nil {
			panic(err)
		}

		res, err := http.Post(fullnodeUrl+"/transaction_batch", "application/json",
			bytes.NewBuffer(jsonData))

		if err != nil {
			panic(err.Error())
		}

		if isResError(res) {
			panic(res.Status)
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			panic(err.Error())
		}
		err = json.Unmarshal(body, &data)
		if err != nil {
			panic(err.Error())
		}
	}

	if includeUnindexed {
		log.Printf("[getTransactions][Getting unindexed transactions]\n")
		res, err := http.Get(fullnodeUrl + "/transaction/none-indexed/batch")

		if err != nil {
			panic(err)
		}

		if isResError(res) {
			panic(res.Status)
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			panic(err.Error())
		}

		var unindexedData []dto.TransactionResponse
		err = json.Unmarshal(body, &unindexedData)
		if err != nil {
			panic(err.Error())
		}
		data = append(data, unindexedData...)
	}
	return data
}

func (service *transactionService) getTransactionsByHash(hashArray []string, fullnodeUrl string) (txs []dto.TransactionResponse, err error) {
	values := map[string][]string{"transactionHashes": hashArray}
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

func (service *transactionService) insertBaseTransactionsInputsOutputs(m map[string]txBuilder, tx *gorm.DB) error {
	var ibtToBeSaved []*entities.InputBaseTransaction
	var rbtToBeSaved []*entities.ReceiverBaseTransaction
	var ffbtToBeSaved []*entities.FullnodeFeeBaseTransaction
	var nfbtToBeSaved []*entities.NetworkFeeBaseTransaction
	for _, value := range m {
		txId := value.DbTx.ID
		for _, baseTransaction := range value.Tx.BaseTransactionsRes {
			switch baseTransaction.Name {
			case "IBT":
				ibt := entities.NewInputBaseTransaction(&baseTransaction, txId)
				ibtToBeSaved = append(ibtToBeSaved, ibt)
			case "RBT":
				rbt := entities.NewReceiverBaseTransaction(&baseTransaction, txId)
				rbtToBeSaved = append(rbtToBeSaved, rbt)
			case "FFBT":
				ffbt := entities.NewFullnodeFeeBaseTransaction(&baseTransaction, txId)
				ffbtToBeSaved = append(ffbtToBeSaved, ffbt)
			case "NFBT":
				nfbt := entities.NewNetworkFeeBaseTransaction(&baseTransaction, txId)
				nfbtToBeSaved = append(nfbtToBeSaved, nfbt)
			default:
				fmt.Println("Unknown base transaction name: ", baseTransaction.Name)
			}
		}

	}
	if len(ibtToBeSaved) > 0 {
		if err := tx.Omit("CreateTime", "UpdateTime").Create(&ibtToBeSaved).Error; err != nil {
			log.Println(err)
			return err
		}
	}

	if len(rbtToBeSaved) > 0 {
		if err := tx.Omit("CreateTime", "UpdateTime").Create(&rbtToBeSaved).Error; err != nil {
			log.Println(err)
			return err
		}
	}
	if len(ffbtToBeSaved) > 0 {
		if err := tx.Omit("CreateTime", "UpdateTime").Create(&ffbtToBeSaved).Error; err != nil {
			log.Println(err)
			return err
		}
	}
	if len(nfbtToBeSaved) > 0 {
		if err := tx.Omit("CreateTime", "UpdateTime").Create(&nfbtToBeSaved).Error; err != nil {
			log.Println(err)
			return err
		}
	}
	return nil

}
