package zplorama

import (
	"errors"

	"github.com/boltdb/bolt"
	"github.com/yosuke-furukawa/json5/encoding/json5"
)

const (
	jobDB string = "jobs.boltdb"
)

func createDB() *bolt.DB {
	db, err := bolt.Open(jobDB, 0600, nil)

	if err != nil {
		panic(err)
	}

	// Make default tables
	db.Update(func(tx *bolt.Tx) error {
		buckets := []string{printjobTable, jobTimeTable}

		for _, bucket := range buckets {
			_, err := tx.CreateBucketIfNotExists([]byte(bucket))

			if err != nil {
				return err
			}
		}

		return nil
	})

	return db
}

// GetRecord pulls from the DB
func GetRecord(database *bolt.DB, record Boltable) error {
	var err error

	database.View(func(tx *bolt.Tx) error {

		bucket := tx.Bucket([]byte(record.Table()))
		recordBytes := bucket.Get([]byte(record.Key()))

		if recordBytes == nil || len(recordBytes) == 0 {
			err = errors.New("Record not found")
		}

		if err == nil {
			err = json5.Unmarshal(recordBytes, record)
		}

		return err
	})

	return err
}

// PutRecord stores a boltable in the database
func PutRecord(database *bolt.DB, record Boltable) error {
	var err error

	database.Update(func(tx *bolt.Tx) error {
		recordBytes, err := json5.Marshal(record)

		if err != nil {
			return err
		}

		bucket := tx.Bucket([]byte(record.Table()))
		err = bucket.Put([]byte(record.Key()), recordBytes)

		return err
	})

	return err
}

// Boltable represents a struct that can be JSON serialized to BoltDB
type Boltable interface {
	Table() string
	Key() string
}
