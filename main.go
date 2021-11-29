package main

import (
	dbprovider "db-sync/db-provider"
	"db-sync/entities"
	"db-sync/service"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	server := gin.Default()
	server.GET("/test", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"message": "OK!!",
		})
	})

	dbprovider.Init()

	// making sure all the app states exists and create them if not
	verifyAppStates()

	transactionService := service.NewTransactionService()
	transactionService.RunSync()

	server.Run(":3000")

}

func verifyAppStates() {
	// TODO find a way to iterate over the enum in the top of the page
	appState := entities.AppState{Name: entities.LastMonitoredTransactionIndex}
	appStateRes := dbprovider.DB.Where("name = ?", entities.LastMonitoredTransactionIndex).FirstOrCreate(&appState)
	if appStateRes.Error != nil {
		panic(appStateRes.Error)
	}
}
