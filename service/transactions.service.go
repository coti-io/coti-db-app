package service

import (
	"bytes"
	"db-sync/dto"
	"db-sync/entities"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BaseTransactionName string

const (
	IBT  BaseTransactionName = "IBT"
	FFBT                     = "FFBT"
	NFBT                     = "NFBT"
	RBT                      = "RBT"
)

type TransactionService interface {
	RunSync()
}
type transactionService struct {
	db            *gorm.DB
	isSyncRunning bool
}

type txBuilder struct {
	Tx   dto.TransactionResponse
	DbTx *entities.BaseTransaction
}

func NewTransactionService(db *gorm.DB) TransactionService {
	return &transactionService{db, false}
}

// TODO: handle all errors by chanels
func (service *transactionService) RunSync() {

	// 1) sync new transactions
	// 	  gets new indexed transactions and puts it in the db
	// 2) update unverified  transactions
	// 	  get all transactions with status that is not confirmed and check if they are and update

	// flag if this one is already running and dont activate if true
	if service.isSyncRunning {
		return
	}
	service.isSyncRunning = true
	// run sync tasks
	go service.syncNewTransactions()
	go service.monitorTransactions()

}

func (service *transactionService) syncNewTransactions() {
	var maxTransactionsInSync int64 = 3000 // inthe future replace by 1000 and export to config
	var includeUnindexed = false
	// when slice was less then 1000 once replace to the other method that gets unidexed onse as well
	iteration := 0
	for {
		dtStart := time.Now()
		fmt.Println("[syncNewTransactions][iteration start] " + strconv.Itoa(iteration))
		tx := service.db.Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()
		if err := tx.Error; err != nil {
			return
		}
		// get last monitored index - we need to consider updated transaction status.. is transaction that are not approved assigned an index?
		var appState entities.AppState
		tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("name = ?", entities.LastMonitoredTransactionIndex).First(&appState)
		var lastMonitoredIndex int64
		if appState.Value == "" {
			lastMonitoredIndex = 0
		} else {
			lastMonitoredIndexInt, err := strconv.Atoi(appState.Value)
			if err != nil {
				panic(err)
			}
			lastMonitoredIndex = int64(lastMonitoredIndexInt)
		}
		// get the tip
		tipDto := service.getTip()
		tipIndex := tipDto.LastIndex
		startingIndex := lastMonitoredIndex
		if startingIndex != 0 {
			startingIndex += 1
		}
		// making sure we dont handle too much in one iteration
		endingIndex := lastMonitoredIndex
		if tipIndex > int64(startingIndex)+int64(maxTransactionsInSync) {
			endingIndex += maxTransactionsInSync
		} else {
			endingIndex = int64(tipIndex) + 1
			includeUnindexed = true
		}
		transactions := service.getTransactions(int64(startingIndex), int64(endingIndex), includeUnindexed)
		if len(transactions) > 0 {
			// get all the transactions hash
			txHashArray := []interface{}{}
			for _, tx := range transactions {
				txHashArray = append(txHashArray, tx.Hash)
			}
			// find recoreds with a tx hash like the one we got and filter them from the array
			var dbTransactionsRes []entities.BaseTransaction
			tx.Where("hash IN (?"+strings.Repeat(",?", len(txHashArray)-1)+")", txHashArray...).Find(&dbTransactionsRes)

			filteredTransactions := []dto.TransactionResponse{}

			largestIndex := 0
			for _, tx := range transactions {
				exists := false
				for _, dbTx := range dbTransactionsRes {
					if dbTx.Hash == tx.Hash {
						exists = true
					}
				}
				if largestIndex < int(tx.Index) {
					largestIndex = int(tx.Index)
				}
				if !exists {
					filteredTransactions = append(filteredTransactions, tx)
				}
			}
			if len(filteredTransactions) > 0 {
				// prepare all the transaction to be saved
				baseTransactionsToBeSaved := []*entities.BaseTransaction{}
				m := map[string]txBuilder{}
				for _, tx := range filteredTransactions {
					dbtx := entities.NewBaseTransaction(&tx)
					baseTransactionsToBeSaved = append(baseTransactionsToBeSaved, dbtx)
					m[dbtx.Hash] = txBuilder{tx, dbtx}
				}

				// save all of them
				if err := tx.Create(&baseTransactionsToBeSaved).Error; err != nil {
					tx.Rollback()
					log.Println(err)
					return
				}

				if largestIndex < int(lastMonitoredIndex) {
					largestIndex = int(lastMonitoredIndex)
				}
				appState.Value = strconv.Itoa(int(largestIndex))

				if err := tx.Save(&appState).Error; err != nil {
					tx.Rollback()
					log.Println(err)
					return
				}
				if err := service.insertBaseTransactionsInputsOutputs(m, tx); err != nil {
					tx.Rollback()
					log.Println(err)
					return
				}

			}

		}
		tx.Commit()
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

func (service *transactionService) monitorTransactions() {
	iteration := 0
	for {
		dtStart := time.Now()
		fmt.Println("[monitorTransactions][iteration start] " + strconv.Itoa(iteration))
		// get all transaction that have index or with status attoched to dag from db
		var dbTransactions []entities.BaseTransaction
		service.db.Where("'index' IS NULL OR trustChainConsensus = 0").Find(&dbTransactions)
		m := map[string]entities.BaseTransaction{}
		hashArray := []string{}
		for _, tx := range dbTransactions {
			m[tx.Hash] = tx
			hashArray = append(hashArray, tx.Hash)
		}
		// get the transactions from the node
		transactions := service.getTransactionsByHash(hashArray)
		// update the transactions
		var transactionToSave []*entities.BaseTransaction
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
			result := service.db.Save(&transactionToSave)

			log.Println(result)
		}

		fmt.Println("[monitorTransactions][iteration end] " + strconv.Itoa(iteration))
		dtEnd := time.Now()
		diff := dtEnd.Sub(dtStart)
		diffInSeconds := diff.Seconds()
		if diffInSeconds < 5 && diffInSeconds > 0 {
			timeDurationToSleep := time.Duration(float64(5) - diffInSeconds)
			fmt.Println("[syncNewTransactions][sleeping for] ", timeDurationToSleep)
			time.Sleep(timeDurationToSleep * time.Second)

		}
	}
}

func (service *transactionService) getTransactions(startingIndex int64, endingIndex int64, includeUnindexed bool) []dto.TransactionResponse {
	values := map[string]string{"startingIndex": strconv.FormatInt(int64(startingIndex), 10), "endingIndex": strconv.FormatInt(int64(endingIndex), 10)}
	json_data, err := json.Marshal(values)

	if err != nil {
		log.Fatal(err)
	}

	res, err := http.Post("https://testnet-staging-fullnode1.coti.io/transaction_batch", "application/json",
		bytes.NewBuffer(json_data))

	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err.Error())
	}

	var data []dto.TransactionResponse
	json.Unmarshal([]byte(body), &data)
	if includeUnindexed {
		res, err := http.Get("https://testnet-staging-fullnode1.coti.io/transaction/none-indexed/batch")

		if err != nil {
			log.Fatal(err)
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			panic(err.Error())
		}

		var unindexedData []dto.TransactionResponse
		json.Unmarshal([]byte(body), &unindexedData)
		data = append(data, unindexedData...)
	}
	return data
}

func (service *transactionService) getTransactionsByHash(hashArray []string) []dto.TransactionResponse {
	values := map[string][]string{"transactionHashes": hashArray}
	json_data, err := json.Marshal(values)

	if err != nil {
		log.Fatal(err)
	}

	res, err := http.Post("https://testnet-staging-fullnode1.coti.io/transaction/multiple", "application/json",
		bytes.NewBuffer(json_data))

	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err.Error())
	}

	var data []dto.TransactionResponse
	json.Unmarshal([]byte(body), &data)
	return data
}

func (service *transactionService) getTip() dto.TransactionsIndexTip {

	res, err := http.Get("https://testnet-staging-fullnode1.coti.io/transaction/lastIndex")

	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err.Error())
	}

	var data dto.TransactionsIndexTip
	json.Unmarshal([]byte(body), &data)
	return data
}

func (service *transactionService) insertBaseTransactionsInputsOutputs(m map[string]txBuilder, tx *gorm.DB) error {
	ibtToBeSaved := []*entities.BaseTransactionsInputs{}
	rbtToBeSaved := []*entities.BaseTransactionsReceivers{}
	ffbtToBeSaved := []*entities.BaseTransactionFNF{}
	nfbtToBeSaved := []*entities.BaseTransactionsNF{}
	for _, value := range m {
		txId := value.DbTx.ID
		for _, baseTransaction := range value.Tx.BaseTransactionsRes {
			switch baseTransaction.Name {
			case "IBT":
				// create IBT record
				ibt := entities.NewBaseTransactionsInputs(&baseTransaction, txId)
				ibtToBeSaved = append(ibtToBeSaved, ibt)
				// service.db.Create(&ibt)
			case "RBT":
				// create RBT record
				rbt := entities.NewBaseTransactionsReceivers(&baseTransaction, txId)
				rbtToBeSaved = append(rbtToBeSaved, rbt)
				// service.db.Create(&rbt)
			case "FFBT":
				// create FFBT record
				ffbt := entities.NewBaseTransactionFNF(&baseTransaction, txId)
				ffbtToBeSaved = append(ffbtToBeSaved, ffbt)
				// service.db.Create(&ffbt)
			case "NFBT":
				// create NFBT record
				nfbt := entities.NewBaseTransactionNF(&baseTransaction, txId)
				nfbtToBeSaved = append(nfbtToBeSaved, nfbt)
				// service.db.Create(&nfbt)
			default:
				fmt.Println("Unknown base transaction name: ", baseTransaction.Name)
			}
		}

	}
	if err := tx.Create(&ibtToBeSaved).Error; err != nil {
		tx.Rollback()
		log.Println(err)
		return err
	}

	if err := tx.Create(&rbtToBeSaved).Error; err != nil {
		tx.Rollback()
		log.Println(err)
		return err
	}
	if err := tx.Create(&ffbtToBeSaved).Error; err != nil {
		tx.Rollback()
		log.Println(err)
		return err
	}
	if err := tx.Create(&nfbtToBeSaved).Error; err != nil {
		tx.Rollback()
		log.Println(err)
		return err
	}
	return nil

}