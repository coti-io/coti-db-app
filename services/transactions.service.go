package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	dbprovider "github.com/coti-io/coti-db-app/db-provider"
	"github.com/coti-io/coti-db-app/dto"
	"github.com/coti-io/coti-db-app/entities"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var once sync.Once

type BaseTransactionName string

const (
	IBT  BaseTransactionName = "IBT"
	FFBT                     = "FFBT"
	NFBT                     = "NFBT"
	RBT                      = "RBT"
)

type TransactionService interface {
	RunSync()
	GetTip() dto.TransactionsIndexTip
	GetLastIterationTip() int64
}
type transactionService struct {
	fullnodeUrl           string
	isSyncRunning         bool
	lastIterationIndexTip int64
}

type txBuilder struct {
	Tx   dto.TransactionResponse
	DbTx *entities.Transaction
}

var instance *transactionService

// NewTransactionService we made this one a singleton because it has a state
func NewTransactionService() TransactionService {
	once.Do(func() {

		instance = &transactionService{os.Getenv("FULLNODE_URL"), false, 0}
	})
	return instance
}

// RunSync TODO: handle all errors by channels
func (service *transactionService) RunSync() {

	// 1) sync new transactions
	// 	  gets new indexed transactions and puts it in the db
	// 2) update unverified  transactions
	// 	  get all transactions with status that is not confirmed and check if they are and update

	// flag if this one is already running and don't activate if true
	if service.isSyncRunning {
		return
	}
	service.isSyncRunning = true
	// run sync tasks
	go service.syncNewTransactions()
	go service.monitorTransactions()

}

func (service *transactionService) syncNewTransactions() {
	var maxTransactionsInSync int64 = 3000 // in the future replace by 1000 and export to config
	var includeUnindexed = false
	// when slice was less than 1000 once replace to the other method that gets un-indexed ones as well
	iteration := 0
	for {
		dtStart := time.Now()
		fmt.Println("[syncNewTransactions][iteration start] " + strconv.Itoa(iteration))
		service.syncTransactionsIteration(maxTransactionsInSync, &includeUnindexed)
		fmt.Println("[syncNewTransactions][iteration end] " + strconv.Itoa(iteration))
		dtEnd := time.Now()
		diff := dtEnd.Sub(dtStart)
		diffInSeconds := diff.Seconds()
		if diffInSeconds < 10 && diffInSeconds > 0 && includeUnindexed {
			timeDurationToSleep := time.Duration(float64(10) - diffInSeconds)
			fmt.Println("[syncNewTransactions][sleeping for] ", timeDurationToSleep)
			time.Sleep(timeDurationToSleep * time.Second)

		}
		iteration += 1
	}

}

func (service *transactionService) syncTransactionsIteration(maxTransactionsInSync int64, includeUnindexed *bool) {
	tx := dbprovider.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		return
	}
	// get last monitored index - we need to consider updated transaction status. Is transaction that are not approved assigned an index?
	var appState entities.AppState
	tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("name = ?", entities.LastMonitoredTransactionIndex).First(&appState)
	var lastMonitoredIndex int64
	if appState.Value == "" {
		lastMonitoredIndex = -1
	} else {
		lastMonitoredIndexInt, err := strconv.Atoi(appState.Value)
		if err != nil {
			panic(err)
		}
		lastMonitoredIndex = int64(lastMonitoredIndexInt)
	}
	// get the tip
	tipDto := service.GetTip()
	tipIndex := tipDto.LastIndex
	service.lastIterationIndexTip = tipIndex
	startingIndex := lastMonitoredIndex + 1

	// making sure we don't handle too much in one iteration
	var endingIndex int64
	if tipIndex > startingIndex+maxTransactionsInSync {
		endingIndex = startingIndex + maxTransactionsInSync - 1
	} else {
		endingIndex = tipIndex
		*includeUnindexed = true
	}
	var includeIndexed = true
	if startingIndex > endingIndex {
		includeIndexed = false
	}
	transactions := service.getTransactions(startingIndex, endingIndex, includeIndexed, *includeUnindexed)
	if len(transactions) > 0 {
		// get all the transactions hash
		var txHashArray []interface{}
		for _, tx := range transactions {
			txHashArray = append(txHashArray, tx.Hash)
		}
		// find records with a tx hash like the one we got and filter them from the array
		var dbTransactionsRes []entities.Transaction
		tx.Where("hash IN (?"+strings.Repeat(",?", len(txHashArray)-1)+")", txHashArray...).Find(&dbTransactionsRes)

		var filteredTransactions []dto.TransactionResponse

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
			// prepare all the transactions to be saved
			var baseTransactionsToBeSaved []*entities.Transaction
			m := map[string]txBuilder{}
			for _, tx := range filteredTransactions {
				dbTx := entities.NewTransaction(&tx)
				baseTransactionsToBeSaved = append(baseTransactionsToBeSaved, dbTx)
				m[dbTx.Hash] = txBuilder{tx, dbTx}
			}

			// save all of them
			if err := tx.Omit("CreateTime", "UpdateTime").Create(&baseTransactionsToBeSaved).Error; err != nil {
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
		if int64(largestIndex) > lastMonitoredIndex {
			appState.Value = strconv.Itoa(largestIndex)

			if err := tx.Omit("CreateTime", "UpdateTime").Save(&appState).Error; err != nil {
				tx.Rollback()
				log.Println(err)
				return
			}
		}
	}

	tx.Commit()
}

func (service *transactionService) monitorTransactions() {
	iteration := 0
	for {
		dtStart := time.Now()
		fmt.Println("[monitorTransactions][iteration start] " + strconv.Itoa(iteration))
		// get all transaction that have index or with status attached to dag from db
		var dbTransactions []entities.Transaction
		dbprovider.DB.Where("'index' IS NULL OR trustChainConsensus = 0").Find(&dbTransactions)
		m := map[string]entities.Transaction{}
		var hashArray []string
		for _, tx := range dbTransactions {
			m[tx.Hash] = tx
			hashArray = append(hashArray, tx.Hash)
		}
		// get the transactions from the node
		transactions := service.getTransactionsByHash(hashArray)
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
			result := dbprovider.DB.Omit("CreateTime", "UpdateTime").Save(&transactionToSave)

			log.Println(result)
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
		iteration += 1
	}
}

func (service *transactionService) getTransactions(startingIndex int64, endingIndex int64, includeIndexed bool, includeUnindexed bool) []dto.TransactionResponse {
	var data []dto.TransactionResponse

	if includeIndexed {
		log.Printf("[getTransactions][Getting transactions from index %d to index %d]\n", startingIndex, endingIndex)
		values := map[string]string{"startingIndex": strconv.FormatInt(startingIndex, 10), "endingIndex": strconv.FormatInt(endingIndex, 10)}
		jsonData, err := json.Marshal(values)

		if err != nil {
			log.Fatal(err)
		}

		res, err := http.Post(service.fullnodeUrl+"/transaction_batch", "application/json",
			bytes.NewBuffer(jsonData))

		if err != nil {
			log.Fatal(err)
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			panic(err.Error())
		}
		json.Unmarshal(body, &data)
	}

	if includeUnindexed {
		log.Printf("[getTransactions][Getting unindexed transactions]\n")
		res, err := http.Get(service.fullnodeUrl + "/transaction/none-indexed/batch")

		if err != nil {
			log.Fatal(err)
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			panic(err.Error())
		}

		var unindexedData []dto.TransactionResponse
		json.Unmarshal(body, &unindexedData)
		data = append(data, unindexedData...)
	}
	return data
}

func (service *transactionService) getTransactionsByHash(hashArray []string) []dto.TransactionResponse {
	values := map[string][]string{"transactionHashes": hashArray}
	jsonData, err := json.Marshal(values)

	if err != nil {
		log.Fatal(err)
	}

	res, err := http.Post(service.fullnodeUrl+"/transaction/multiple", "application/json",
		bytes.NewBuffer(jsonData))

	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err.Error())
	}

	var data []dto.TransactionResponse
	json.Unmarshal(body, &data)
	return data
}

func (service *transactionService) GetTip() dto.TransactionsIndexTip {

	res, err := http.Get(service.fullnodeUrl + "/transaction/lastIndex")

	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err.Error())
	}

	var data dto.TransactionsIndexTip
	json.Unmarshal(body, &data)
	return data
}

func (service *transactionService) GetLastIterationTip() int64 {
	return service.lastIterationIndexTip
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
