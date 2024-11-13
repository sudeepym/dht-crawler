package dht

import (
	"bytes"
	"fmt"
	"log"
	"strconv"

	"github.com/boltdb/bolt"
	"github.com/jackpal/bencode-go"
)

func DeleteInfohash(infohash string) error {
	db, err := bolt.Open("torrent.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	return db.Update(func(tx *bolt.Tx) error {
		// Create or get the "Metadata" bucket
		bucket, err := tx.CreateBucketIfNotExists([]byte("Metadata"))
		if err != nil {
			return fmt.Errorf("failed to create bucket: %v", err)
		}
		// Store the metadata with the infohash as the key
		return bucket.Delete([]byte(infohash))
	})
}

func CheckInfohashExists(infohash string) bool {
	// Open the BoltDB database file
	// db, err := bolt.Open("torrent.db", 0600, nil)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer db.Close()

	var exists bool

	// Check if the infohash key exists in the "Metadata" bucket
	err := db.View(func(tx *bolt.Tx) error {
		// Open the bucket
		bucket := tx.Bucket([]byte("Metadata"))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		// Get the value associated with the infohash key
		metadata := bucket.Get([]byte(infohash))
		exists = metadata != nil // If metadata is nil, the key doesn't exist
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	return exists
}


func ShowMetadataForInfohash(infohash string)([]byte){
	// Open the BoltDB database file

	var ret []byte
	// Read the metadata associated with the infohash
	err := db.View(func(tx *bolt.Tx) error {
		// Open the bucket (substitute "MetadataBucket" with your actual bucket name)
		bucket := tx.Bucket([]byte("Metadata"))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}
		// Retrieve metadata by the infohash key
		metadata := bucket.Get([]byte(infohash))
		if metadata == nil {
			fmt.Println("No metadata found for infohash:", infohash)
			return nil
		}
		// Convert metadata to string and display it (assuming metadata is stored as bytes)
		// fmt.Println("Metadata for infohash:", infohash)
		// fmt.Println(string(metadata)) // Or handle it as needed (e.g., unmarshal if JSON or bencode)
		ret = make([]byte,len(metadata))
		copy(ret,metadata)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return ret
}

func ShowInfohashes()([]string){
	// Open the BoltDB database file
	db, err := bolt.Open("torrent.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var infohashes []string
	// Read the metadata associated with the infohash
	err = db.View(func(tx *bolt.Tx) error {
		// Open the bucket (substitute "MetadataBucket" with your actual bucket name)
		bucket := tx.Bucket([]byte("Metadata"))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		// Iterate over each key-value pair in the bucket
		i:=1
		err = bucket.ForEach(func(k, v []byte) error {
			infohash := string(k)
			metadata := string(v) // Assuming metadata is stored as a string; adapt as needed
			_ = metadata
			// Print or process each infohash-metadata pair
			infohashes = append(infohashes, infohash)
			// fmt.Printf("%d: %s\n",i, infohash)
			i++
			return nil
		})
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return infohashes
}

type File struct {
	Length	int	`bencode:"length"`
	Path	[]string	`bencode:"path"`
}

type MetaData struct {
	Files	[]File	`bencode:"files"`
}

type Pieces string

func ParseMetadata(metadata []byte) (string, []string) {
	// Initialize return values
	var name string
	var files []string

	// Find the "4:name" key in the metadata
	metaNameIndex := bytes.Index(metadata, []byte("4:name"))
	if metaNameIndex != -1 {
		// Extract the length of the name
		colonIndex := metaNameIndex + len("4:name")
		nameLengthStart := colonIndex
		nameLengthEnd := bytes.IndexByte(metadata[nameLengthStart:], ':') + nameLengthStart
		nameLengthStr := string(metadata[nameLengthStart:nameLengthEnd])

		// Convert the length to an integer
		nameLength, err := strconv.Atoi(nameLengthStr)
		if err != nil {
			// Return empty if there's an error with parsing the length
			return name, files
		}

		// Extract the name based on the length
		nameStart := nameLengthEnd + 1
		name = string(metadata[nameStart : nameStart+nameLength])
	}

	// Attempt to parse the "files" section
	metaFilesIndexStart := bytes.Index(metadata, []byte("5:files"))
	if metaFilesIndexStart != -1 {
		metaFilesIndexEnd := bytes.Index(metadata, []byte("4:name"))
		if metaFilesIndexEnd == -1{
			return name, files
		}
		metaFiles := metadata[metaFilesIndexStart:metaFilesIndexEnd]
		filesData := make([]byte,len(metaFiles))
		copy(filesData,metaFiles)
		// fmt.Println(string(filesData))
		filesData = append([]byte("d"),filesData...)
		filesData = append(filesData, byte('e'))
		var metaFilesDict MetaData
		if err := bencode.Unmarshal(bytes.NewReader(filesData), &metaFilesDict); err == nil {
			for _, file := range metaFilesDict.Files {
				files = append(files, file.Path...)
			}
		}
	}

	return name, files
}
