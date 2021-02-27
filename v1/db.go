package zplorama

import "github.com/boltdb/bolt"

const (
	printjobTable string = "printjobs"
	jobTimeTable         = "jobtimes"
)

func createDB() *bolt.DB {
	db, err := bolt.Open(printjobTable, 0600, nil)

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
