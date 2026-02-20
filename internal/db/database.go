package db

import (
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// NewDatabase initializes a new GORM database connection and runs auto-migrations.
func NewDatabase(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	log.Println("Running database migrations...")
	// Auto-migrate the schema
	err = db.AutoMigrate(
		&Node{},
		&Deployment{},
		&ContainerInstance{},
		&RegistryCredentials{},
		&Network{},
	)
	if err != nil {
		return nil, err
	}

	log.Println("Database connection established and migrations completed.")
	return db, nil
}
