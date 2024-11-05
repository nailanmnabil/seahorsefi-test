package pkg

import (
	"log"
	"strconv"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func GetDbConn(stringConn, dbMinConn, dbMaxConn string) *gorm.DB {
	db, err := gorm.Open(postgres.Open(stringConn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v\n", err)
	}
	db = db.Session(&gorm.Session{SkipDefaultTransaction: true})

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get DB from GORM: %v\n", err)
	}

	minConn, err := strconv.Atoi(dbMinConn)
	if err != nil {
		log.Fatalf("Failed to convert dbMinConn: %v\n", err)
	}
	maxConn, err := strconv.Atoi(dbMaxConn)
	if err != nil {
		log.Fatalf("Failed to convert dbMaxConn: %v\n", err)
	}

	sqlDB.SetMaxIdleConns(minConn)
	sqlDB.SetMaxOpenConns(maxConn)

	return db
}
