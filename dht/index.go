package dht

import (
	"encoding/binary"
	"fmt"
	"log"
	"strings"
	"unicode"

	"github.com/boltdb/bolt"
)


// Converts an integer score to a byte slice for storing in BoltDB
func intToBytes(score int) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(score))
	return buf
}

func Index(infohash, name string, files []string) {
	scoreMap := make(map[string]int)
	// db, err := bolt.Open("torrent.db", 0600, nil)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer db.Close()

	splitter := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	}

	
	// Calculate scores based on the name, if provided
	if name != "" {
		for _, token := range strings.FieldsFunc(name, splitter) {
			scoreMap[strings.ToLower(token)] += 20
		}
	}

	// Calculate scores based on file paths, if files are provided
	if len(files) > 0 {
		for _, file := range files {
			for _, token := range strings.FieldsFunc(file, splitter) {
				scoreMap[strings.ToLower(token)] += 10
			}
		}
	}

	// Skip storing if no valid words were found
	if len(scoreMap) == 0 {
		fmt.Println("No tokens found for indexing for infohash :",infohash)
		return
	}

	// Store each word in the "Search" bucket with infohash-score pairs in sub-buckets
	err := db.Batch(func(tx *bolt.Tx) error {
		// Create or get the top-level "Search" bucket
		searchBucket, err := tx.CreateBucketIfNotExists([]byte("Search"))
		if err != nil {
			return fmt.Errorf("failed to create 'Search' bucket: %v", err)
		}

		for word, score := range scoreMap {
			// Create or get a sub-bucket for each word within "Search"
			wordBucket, err := searchBucket.CreateBucketIfNotExists([]byte(word))
			if err != nil {
				return fmt.Errorf("failed to create word bucket '%s': %v", word, err)
			}

			// Store the infohash as the key and the score (converted to bytes) as the value
			if err := wordBucket.Put([]byte(infohash), intToBytes(score)); err != nil {
				return fmt.Errorf("failed to put infohash-score pair in word bucket '%s': %v", word, err)
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
