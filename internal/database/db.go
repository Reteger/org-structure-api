package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(databaseConnectionString string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(databaseConnectionString), &gorm.Config{
		SkipDefaultTransaction: true,
	})
}
