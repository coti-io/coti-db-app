package main

import (
	"db-sync/entities"
	"db-sync/service"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
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
	dbName := os.Getenv("DB_NAME")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbPort := os.Getenv("DB_PORT")
	dbHost := os.Getenv("DB_HOST")
	dbConnectionString := dbUser + `:` + dbPassword + `@tcp(` + dbHost + `:` + dbPort + `)/` + dbName + `?charset=utf8&parseTime=True&loc=Local`
	db, dbError := gorm.Open(mysql.New(mysql.Config{
		DSN:                       dbConnectionString, // data source name
		DefaultStringSize:         500,                // default size for string fields
		DisableDatetimePrecision:  true,               // disable datetime precision, which not supported before MySQL 5.6
		DontSupportRenameIndex:    true,               // drop & create when rename index, rename index not supported before MySQL 5.7, MariaDB
		DontSupportRenameColumn:   true,               // `change` when rename column, rename column not supported before MySQL 8, MariaDB
		SkipInitializeWithVersion: false,              // auto configure based on currently MySQL version
	}), &gorm.Config{})

	if dbError != nil {
		panic("failed to connect database")
	}
	// making sure all the app states exists and create them if not
	verifyAppStates(db)

	transactionService := service.NewTransactionService(db)
	transactionService.RunSync()

	server.Run(":3000")

}

func verifyAppStates(db *gorm.DB) {
	// TODO find a way to iterate over the enum in the top of the page
	appState := entities.AppState{Name: entities.LastMonitoredTransactionIndex}
	appStateRes := db.Where("name = ?", entities.LastMonitoredTransactionIndex).FirstOrCreate(&appState)
	if appStateRes.Error != nil {
		panic(appStateRes.Error)
	}
}
