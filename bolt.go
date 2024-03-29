package main

import (
	"fmt"
	"log"

	"github.com/boltdb/bolt"
)

//dbUpdate will put key/values in a specified db, and in a specified bucket.
func dbUpdate(db *bolt.DB, bucketName string, key string, value string) error {
	fmt.Println("*************************dbUpdate*1")
	err := db.Update(func(tx *bolt.Tx) error {
		fmt.Println("*************************dbUpdate*2")
		//Create a bucket
		bu, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return fmt.Errorf("error: CreateBuckerIfNotExists failed: %v", err)
		}
		fmt.Println("*************************dbUpdate*3")

		//Put a value into the bucket.
		if err := bu.Put([]byte(key), []byte(value)); err != nil {
			return err
		}
		fmt.Println("*************************dbUpdate*4")

		//If all was ok, we should return a nil for a commit to happen. Any error
		// returned will do a rollback.
		return nil
	})
	return err
}

//dbViewSingle will look up the value for a specified key.
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

//dbViewAll will get all the key/values from a specified bucket.
func dbViewAll(db *bolt.DB, bucketName string) (map[string]string, error) {
	//Map for returning the key values in the db.
	m := make(map[string]string)

	err := db.View(func(tx *bolt.Tx) error {

		bu := tx.Bucket([]byte(bucketName))

		//Check if tx.Bucket returns nil, and return if nil,
		// if it was nil and we did continue it will panic
		// on the first use of the bucket, since it does not exist.
		if bu == nil {
			log.Println("error: bucket does not exist: ", bu)
			return fmt.Errorf("error: bucket does not exist: value : %v", bu)
		}

		//create a cursor, go to the first key in the db,
		// then iterate all key's with next.
		cursor := bu.Cursor()
		cursor.First()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			m[string(k)] = string(v)
			//fmt.Printf("key=%s, value=%s\n", k, v)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error: viewing all items in db: %v", err)
	}

	return m, err
}
