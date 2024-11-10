package main

import (
	"dht-crawler/dht"
	// "fmt"
	// "log"
	// "github.com/boltdb/bolt"
)

func main() {
	// // Open the BoltDB database file
	// db, err := bolt.Open("torrent.db", 0600, nil)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer db.Close()

	// // Read the metadata associated with the infohash
	// err = db.View(func(tx *bolt.Tx) error {
	// 	// Open the bucket (substitute "MetadataBucket" with your actual bucket name)
	// 	bucket := tx.Bucket([]byte("Metadata"))
	// 	if bucket == nil {
	// 		return fmt.Errorf("bucket not found")
	// 	}
		
	// 	infohash := "3edb2055e5cc68e8420a2388755d964e4bfd70a1"
	// 	// Retrieve metadata by the infohash key
	// 	metadata := bucket.Get([]byte(infohash))
	// 	if metadata == nil {
	// 		fmt.Println("No metadata found for infohash:", infohash)
	// 		return nil
	// 	}
	// 	// Convert metadata to string and display it (assuming metadata is stored as bytes)
	// 	fmt.Println("Metadata for infohash:", infohash)
	// 	fmt.Println(string(metadata)) // Or handle it as needed (e.g., unmarshal if JSON or bencode)

	// 	// Iterate over each key-value pair in the bucket
	// 	err = bucket.ForEach(func(k, v []byte) error {
	// 		infohash := string(k)
	// 		metadata := string(v) // Assuming metadata is stored as a string; adapt as needed
	// 		_ = metadata
	// 		// Print or process each infohash-metadata pair
	// 		fmt.Printf("Infohash: %s\n", infohash)
	// 		return nil
	// 	})
	// 	return nil
	// })
	// if err != nil {
	// 	log.Fatal(err)
	// }

	dht.CrawlDHT()
}
