package dbProvider

import (
	"github.com/coti-io/coti-db-app/entities"
	"gorm.io/gorm/logger"
	"os"
	"time"

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
	if os.Getenv("MIGRATE_DB") == "true" {
		migrateDb(dbName, dbUser, dbHost, dbPort, dbPassword)
	}

	dbConnectionString := dbUser + `:` + dbPassword + `@tcp(` + dbHost + `:` + dbPort + `)/` + dbName + `?charset=utf8&parseTime=True&loc=Local`
	db, dbError := gorm.Open(mysql.New(mysql.Config{
		DSN:                       dbConnectionString, // data source name
		DefaultStringSize:         256,                // default size for string fields
		DisableDatetimePrecision:  true,               // disable datetime precision, which not supported before MySQL 5.6
		DontSupportRenameIndex:    true,               // drop & create when rename index, rename index not supported before MySQL 5.7, MariaDB
		DontSupportRenameColumn:   true,               // `change` when rename column, rename column not supported before MySQL 8, MariaDB
		SkipInitializeWithVersion: false,              // auto configure based on currently MySQL version
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})

	if dbError != nil {
		panic("failed to connect database")
	}

	DB = db
}

func migrateDb(dbName string, dbUser string, dbHost string, dbPort string, dbPassword string) {
	dbConnectionWithNoDbString := dbUser + `:` + dbPassword + `@tcp(` + dbHost + `:` + dbPort + `)/`
	db, dbError := gorm.Open(mysql.New(mysql.Config{
		DSN: dbConnectionWithNoDbString, // data source name
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})

	if dbError != nil {
		panic("failed to connect database")
	}
	db = db.Set("gorm:table_options", "CHARSET=utf8 auto_increment=1")

	createDbRes := db.Exec("CREATE DATABASE IF NOT EXISTS " + dbName + " DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci ")
	if createDbRes.Error != nil {
		panic(createDbRes.Error)
	}

	setDbRes := db.Exec("USE " + dbName)
	if setDbRes.Error != nil {
		panic(setDbRes.Error)
	}

	// Migrate the schema
	db.AutoMigrate(&entities.AppState{}, &entities.Transaction{}, &entities.FullnodeFeeBaseTransaction{}, &entities.InputBaseTransaction{}, &entities.NetworkFeeBaseTransaction{}, &entities.ReceiverBaseTransaction{})
	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	dbCloseError := sqlDB.Close()
	if dbCloseError != nil {
		panic(dbCloseError)
	}
}
