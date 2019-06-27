package main

import (
	"fmt"

	"github.com/boltdb/bolt"
)

func dbUpdate(db *bolt.DB, bucketName string, key string, value string) error {
	err := db.Update(func(tx *bolt.Tx) error {
		//Create a bucket
		bu, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return fmt.Errorf("error: CreateBuckerIfNotExists failed: %v", err)
		}

		//Put a value into the bucket.
		if err := bu.Put([]byte(key), []byte(value)); err != nil {
			return err
		}

		//If all was ok, we should return a nil for a commit to happen. Any error
		// returned will do a rollback.
		return nil
	})
	return err
}

func dbViewSingle(db *bolt.DB, bucketName string, key string) (string, error) {
	var value string
	//View is a help function to get values out of the database.
	err := db.View(func(tx *bolt.Tx) error {
		//Open a bucket to get key's and values from.
		bu := tx.Bucket([]byte(bucketName))

		v := bu.Get([]byte(key))
		if len(v) == 0 {
			return fmt.Errorf("info: view: key not found")
		}

		value = string(v)

		return nil
	})

	return value, err

}

func dbViewAll(db *bolt.DB, bucketName string) error {
	err := db.View(func(tx *bolt.Tx) error {
		bu := tx.Bucket([]byte(bucketName))
		cursor := bu.Cursor()

		k, v := cursor.First()
		fmt.Println("first key/value = ", k, v)

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			fmt.Printf("key=%s, value=%s\n", k, v)
		}

		return nil
	})

	return err

}
