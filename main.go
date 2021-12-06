package main

import (
	"log"

	"github.com/coti-io/coti-db-app/controllers"
	dbprovider "github.com/coti-io/coti-db-app/db-provider"
	"github.com/coti-io/coti-db-app/entities"
	service "github.com/coti-io/coti-db-app/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
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

	// run the sync tasks
	transactionService := service.NewTransactionService()
	transactionService.RunSync()

	// register routes
	server.GET("/get-sync-state", controllers.GetSyncState)

	// runs the server
	serverRunError := server.Run(":3000")
	if serverRunError != nil {
		log.Fatal("Server run error")
		return
	}

}

func verifyAppStates() {
	// TODO find a way to iterate over the enum in the top of the page
	appState := entities.AppState{Name: entities.LastMonitoredTransactionIndex}
	appStateRes := dbprovider.DB.Where("name = ?", entities.LastMonitoredTransactionIndex).FirstOrCreate(&appState)
	if appStateRes.Error != nil {
		panic(appStateRes.Error)
	}
}
