package dht

import (
	"encoding/binary"
	"fmt"
	"log"

	"github.com/boltdb/bolt"
)

func bytesToInt(data []byte) int {
	return int(binary.BigEndian.Uint32(data))
}

func CheckIndexing() {
	// Open the BoltDB database
	db, err := bolt.Open("torrent.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// View the contents of the "Search" bucket
	err = db.View(func(tx *bolt.Tx) error {
		// Get the "Search" bucket
		searchBucket := tx.Bucket([]byte("Search"))
		if searchBucket == nil {
			return fmt.Errorf("bucket 'Search' not found")
		}

		// Iterate over each word bucket within "Search"
		return searchBucket.ForEach(func(word, _ []byte) error {
			fmt.Printf("Word: %s\n", word)

			// For each word bucket, get the sub-bucket
			wordBucket := searchBucket.Bucket(word)
			if wordBucket == nil {
				return fmt.Errorf("sub-bucket '%s' not found", word)
			}

			// Iterate over each infohash-score pair in the word bucket
			return wordBucket.ForEach(func(infohash, scoreData []byte) error {
				score := bytesToInt(scoreData)
				fmt.Printf("  Infohash: %s, Score: %d\n", infohash, score)
				return nil
			})
		})
	})
	if err != nil {
		log.Fatal(err)
	}
}