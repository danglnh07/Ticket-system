package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Queries struct {
	DB *gorm.DB
}

func NewQueries() *Queries {
	return &Queries{}
}

func (queries *Queries) ConnectDB(connStr string) error {
	conn, err := gorm.Open(postgres.Open(connStr))
	if err != nil {
		return err
	}

	queries.DB = conn
	return nil
}

func (queries *Queries) AutoMigration() error {
	return queries.DB.AutoMigrate(&Account{}, &Membership{}, &Event{}, &Ticket{}, &Booking{})
}
