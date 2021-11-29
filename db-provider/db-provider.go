package dbprovider

import (
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init() {
	dbName := os.Getenv("DB_NAME")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbPort := os.Getenv("DB_PORT")
	dbHost := os.Getenv("DB_HOST")
	dbConnectionString := dbUser + `:` + dbPassword + `@tcp(` + dbHost + `:` + dbPort + `)/` + dbName + `?charset=utf8&parseTime=True&loc=Local`
	db, dbError := gorm.Open(mysql.New(mysql.Config{
		DSN:                       dbConnectionString, // data source name
		DefaultStringSize:         256,                // default size for string fields
		DisableDatetimePrecision:  true,               // disable datetime precision, which not supported before MySQL 5.6
		DontSupportRenameIndex:    true,               // drop & create when rename index, rename index not supported before MySQL 5.7, MariaDB
		DontSupportRenameColumn:   true,               // `change` when rename column, rename column not supported before MySQL 8, MariaDB
		SkipInitializeWithVersion: false,              // auto configure based on currently MySQL version
	}), &gorm.Config{})

	if dbError != nil {
		panic("failed to connect database")
	}

	// The section below is for initiating db when running the app - to activate we need to remove the db name from the con string and add all the models to the auto migrate

	// createDbRes := db.Exec("CREATE DATABASE IF NOT EXISTS " + dbName)
	// if createDbRes.Error != nil {
	// 	panic(createDbRes.Error)
	// }

	// setDbRes := db.Exec("USE " + dbName)
	// if setDbRes.Error != nil {
	// 	panic(setDbRes.Error)
	// }

	// // Migrate the schema
	// db.AutoMigrate(&User{})
	DB = db
}
