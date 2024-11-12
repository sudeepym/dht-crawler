package dht

import (
	"bytes"
	"fmt"
	"log"

	"github.com/boltdb/bolt"
	"github.com/jackpal/bencode-go"
)

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


func ShowMetadataForInfohash(infohash string){
	// Open the BoltDB database file
	db, err := bolt.Open("torrent.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Read the metadata associated with the infohash
	err = db.View(func(tx *bolt.Tx) error {
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
		fmt.Println("Metadata for infohash:", infohash)
		fmt.Println(string(metadata)) // Or handle it as needed (e.g., unmarshal if JSON or bencode)

		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

func ShowInfohashes(){
	// Open the BoltDB database file
	db, err := bolt.Open("torrent.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

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
			fmt.Printf("%d: %s\n",i, infohash)
			i++
			return nil
		})
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

func ParseMetadata(metadata []byte){
	fmt.Println(string(metadata))
	metaRespIndex := bytes.Index([]byte(metadata), []byte("d5:files"))
	metaResp := metadata[:metaRespIndex]
	metadata = metadata[metaRespIndex:]
	metaFilesIndex := bytes.Index([]byte(metadata), []byte("4:name"))
	metaFiles := make([]byte, metaFilesIndex)
	copy(metaFiles, metadata[:metaFilesIndex])
	metaFiles = append(metaFiles, byte('e'))
	metadata = metadata[metaFilesIndex:]
	metaNameIndex := bytes.Index([]byte(metadata), []byte("6:pieces"))
	metaName := make([]byte, metaNameIndex)
	copy(metaName, metadata[:metaNameIndex])
	metaName = append([]byte("d"),metaName...)
	metaName = append(metaName, byte('e'))
	metadata = metadata[metaNameIndex:]
	
	
	metaPieces := Pieces(metadata)
	_=metaPieces

	var metaRespDict MetaData
	err := bencode.Unmarshal(bytes.NewReader(metaResp), &metaRespDict)
	if err != nil {
		log.Fatal(err)
		return
	}
	if metaRespDict.MsgType != 1 {
		log.Fatal("invalid msg_type in metadata")
	}
	fmt.Println(metaRespDict)

	var metaFilesDict MetaData
	err = bencode.Unmarshal(bytes.NewReader(metaFiles), &metaFilesDict)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(metaFilesDict)
	var metaNameDict MetaData
	err = bencode.Unmarshal(bytes.NewReader(metaName), &metaNameDict)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(metaNameDict)
	
}