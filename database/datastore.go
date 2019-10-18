package database

import (
	"errors"
	"time"

	"github.com/boltdb/bolt"
)

const (
	dbName            = "expense.db"
	defaultBucketName = "default"
)

var (
	errBucketNotFound = errors.New("Bucket not found")
)

//StoreData open the DB connection for storing the value
func storeData(bucketName, key string, value []byte) error {
	// Open the expense.db data file in your current directory.
	// It will be created if it doesn't exist.
	db, err := bolt.Open(dbName, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}
	defer db.Close()

	// store some data
	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return err
		}

		err = bucket.Put([]byte(key), value)
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

//RetriveData open the DB connection for retriving the value
func retriveData(bucketName, key string) (string, error) {
	var val string
	// Open the expense.db data file in your current directory.
	// It will be created if it doesn't exist.
	db, err := bolt.Open(dbName, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return string(val), err
	}
	defer db.Close()

	//retrive data
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return errBucketNotFound
		}
		val = string(bucket.Get([]byte(key)))
		return nil
	})
	return val, err
}

//DeleteBucket deletes the bucket name
func DeleteBucket(bucketName string) error {
	db, err := bolt.Open(dbName, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte(bucketName))
	})
	return err
}
