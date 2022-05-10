package main

import (
	"encoding/csv"
	"fmt"
	"github.com/coti-io/coti-db-app/controllers"
	dbprovider "github.com/coti-io/coti-db-app/db-provider"
	"github.com/coti-io/coti-db-app/dto"
	"github.com/coti-io/coti-db-app/entities"
	service "github.com/coti-io/coti-db-app/services"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"log"
	"os"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	server := gin.Default()

	// Init the db connection
	dbprovider.Init()

	// making sure all the app states exists and create them if not
	verifyAppStates()

	// making sure we have the native currency hash in the db
	verifyNativeCurrencyHash()

	// load cluster stamp if not loaded
	loadClusterStamp()

	// run the sync tasks
	transactionService := service.NewTransactionService()
	transactionService.RunSync()

	// register routes
	server.GET("/get-sync-state", controllers.GetSyncState)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	// runs the server
	serverRunError := server.Run(":" + port)
	if serverRunError != nil {
		log.Fatal("Server run error")
		return
	}

}

func verifyAppStates() {
	// TODO find a way to iterate over the enum in the top of the page
	appStateLastMonitoredTransactionIndex := entities.AppState{Name: entities.LastMonitoredTransactionIndex}
	appStateLastMonitoredTransactionIndexRes := dbprovider.DB.Where("name = ?", entities.LastMonitoredTransactionIndex).FirstOrCreate(&appStateLastMonitoredTransactionIndex)
	if appStateLastMonitoredTransactionIndexRes.Error != nil {
		panic(appStateLastMonitoredTransactionIndexRes.Error)
	}

	appStateIsClusterStampInitialized := entities.AppState{Name: entities.IsClusterStampInitialized}
	appStateIsClusterStampInitializedRes := dbprovider.DB.Where("name = ?", entities.IsClusterStampInitialized).FirstOrCreate(&appStateIsClusterStampInitialized)
	if appStateIsClusterStampInitializedRes.Error != nil {
		panic(appStateIsClusterStampInitializedRes.Error)
	}

	appStateUpdateBalances := entities.AppState{Name: entities.UpdateBalances}
	appStateUpdateBalancesRes := dbprovider.DB.Where("name = ?", entities.UpdateBalances).FirstOrCreate(&appStateUpdateBalances)
	if appStateUpdateBalancesRes.Error != nil {
		panic(appStateUpdateBalancesRes.Error)
	}

	appStateDeleteUnindexedTransactions := entities.AppState{Name: entities.DeleteUnindexedTransactions}
	appStateDeleteUnindexedTransactionsRes := dbprovider.DB.Where("name = ?", entities.DeleteUnindexedTransactions).FirstOrCreate(&appStateDeleteUnindexedTransactions)
	if appStateDeleteUnindexedTransactionsRes.Error != nil {
		panic(appStateDeleteUnindexedTransactionsRes.Error)
	}

	appStateMonitorTransaction := entities.AppState{Name: entities.MonitorTransaction}
	appStateMonitorTransactionRes := dbprovider.DB.Where("name = ?", entities.MonitorTransaction).FirstOrCreate(&appStateMonitorTransaction)
	if appStateMonitorTransactionRes.Error != nil {
		panic(appStateMonitorTransactionRes.Error)
	}
}

func verifyNativeCurrencyHash() {
	nativeCurrencyHash := "e72d2137d5cfcc672ab743bddbdedb4e059ca9d3db3219f4eb623b01"
	nativeCurrency := entities.Currency{Hash: nativeCurrencyHash}
	nativeCurrencyError := dbprovider.DB.Where("hash = ?", nativeCurrencyHash).FirstOrCreate(&nativeCurrency).Error
	if nativeCurrencyError != nil {
		panic(nativeCurrencyError)
	}
}

func loadClusterStamp() {
	// check the app state if cluster stamp initialized for this db
	appStateIsClusterStampInitialized := entities.AppState{}
	err := dbprovider.DB.Where("name = ?", entities.IsClusterStampInitialized).First(&appStateIsClusterStampInitialized).Error
	if err != nil {
		panic(err)
	}

	if appStateIsClusterStampInitialized.Value != "true" {
		nativeCurrencyHash := "e72d2137d5cfcc672ab743bddbdedb4e059ca9d3db3219f4eb623b01"
		nativeCurrency := entities.Currency{}
		nativeCurrencyError := dbprovider.DB.Where("hash = ?", nativeCurrencyHash).First(&nativeCurrency).Error
		if nativeCurrencyError != nil {
			panic(nativeCurrencyError)
		}
		clusterStampFileName := os.Getenv("CLUSTER_STAMP_FILE_NAME")
		csvFile, err := os.Open(clusterStampFileName)
		if err != nil {
			panic(err)
		}
		fmt.Println("Successfully Opened CSV file")
		defer csvFile.Close()

		reader := csv.NewReader(csvFile)

		err = dbprovider.DB.Transaction(func(dbTransaction *gorm.DB) error {
			addressBalances := []entities.AddressBalance{}
			recordsToSaveCounter := 0
			for {
				line, err := reader.Read()
				if err != nil {
					break
				}
				amount, err := decimal.NewFromString(line[1])
				if err != nil {
					return err
				}
				clusterStampData := dto.ClusterStampDataRow{
					Address:    line[0],
					Amount:     amount,
					CurrencyId: nativeCurrency.ID,
				}
				addressBalance := entities.NewAddressBalanceFromClusterStamp(&clusterStampData)

				addressBalances = append(addressBalances, *addressBalance)
				recordsToSaveCounter += 1
				if recordsToSaveCounter == 1000 {
					err = dbTransaction.Save(addressBalances).Error
					if err != nil {
						return err
					}
					addressBalances = []entities.AddressBalance{}
					recordsToSaveCounter = 0
				}

			}
			if len(addressBalances) > 0 {
				err = dbTransaction.Save(addressBalances).Error
				if err != nil {
					return err
				}
			}

			appStateIsClusterStampInitialized.Value = "true"
			err = dbTransaction.Save(appStateIsClusterStampInitialized).Error
			if err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			panic(err)
		}

	}
}
