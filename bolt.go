package main

import (
	"fmt"
	"log"
	"time"

	"github.com/boltdb/bolt"
)

//dbUpdate will put key/values in a specified db, and in a specified bucket.
func dbUpdate(db2 *bolt.DB, bucketName string, key string, value string) error {

	//--
	db, err := bolt.Open(dbName, 0600, &bolt.Options{
		Timeout:  1 * time.Second,
		ReadOnly: false,
	})

	if err != nil {
		fmt.Println("failed to open boltdb")
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))

		return err
	})

	if err != nil {
		fmt.Println("failed to create bucket")
	}

	defer db.Close()

	//--

	fmt.Println("*************************dbUpdate*1")
	err = db.Update(func(tx *bolt.Tx) error {
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
func dbViewSingle(db2 *bolt.DB, bucketName string, key string) (string, error) {
	var value string

	//--
	db, err := bolt.Open(dbName, 0600, &bolt.Options{
		Timeout:  1 * time.Second,
		ReadOnly: false,
	})

	if err != nil {
		fmt.Println("failed to open boltdb")
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))

		return err
	})

	if err != nil {
		fmt.Println("failed to create bucket")
	}

	defer db.Close()

	//--

	//View is a help function to get values out of the database.
	err = db.View(func(tx *bolt.Tx) error {
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
func dbViewAll(db2 *bolt.DB, bucketName string) (map[string]string, error) {
	//Map for returning the key values in the db.
	m := make(map[string]string)
	//--
	db, err := bolt.Open(dbName, 0600, &bolt.Options{
		Timeout:  1 * time.Second,
		ReadOnly: false,
	})

	if err != nil {
		fmt.Println("failed to open boltdb")
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))

		return err
	})

	if err != nil {
		fmt.Println("failed to create bucket")
	}

	defer db.Close()
	//--

	fmt.Println("*************************dbViewAll*1")
	err = db.View(func(tx *bolt.Tx) error {

		fmt.Println("*************************dbViewAll*2")
		bu := tx.Bucket([]byte(bucketName))
		fmt.Println("*************************dbViewAll*3")
		//Check if tx.Bucket returns nil, and return if nil,
		// if it was nil and we did continue it will panic
		// on the first use of the bucket, since it does not exist.
		if bu == nil {
			log.Println("error: bucket does not exist: ", bu)
			return fmt.Errorf("error: bucket does not exist: value : %v", bu)
		}
		fmt.Println("*************************dbViewAll*4")

		//create a cursor, go to the first key in the db,
		// then iterate all key's with next.
		cursor := bu.Cursor()
		fmt.Println("*************************dbViewAll*5")
		cursor.First()
		fmt.Println("*************************dbViewAll*6")
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			m[string(k)] = string(v)
			//fmt.Printf("key=%s, value=%s\n", k, v)
		}
		fmt.Println("*************************dbViewAll*7")
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error: viewing all items in db: %v", err)
	}

	return m, err
}
