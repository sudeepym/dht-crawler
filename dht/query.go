package dht

import (
    "fmt"
    "log"
    "sort"
    "strings"
    "unicode"

    "github.com/boltdb/bolt"
)

func Query(query string) {
    query = strings.ToLower(query)
    // db, err := bolt.Open("torrent.db", 0600, nil)
    // if err != nil {
    //     log.Fatal(err)
    // }
    // defer db.Close()
    scoreMap := make(map[string]int)
    splitter := func(c rune) bool {
        return !unicode.IsLetter(c) && !unicode.IsNumber(c)
    }

    err := db.View(func(tx *bolt.Tx) error {
        searchBucket := tx.Bucket([]byte("Search"))
        if searchBucket == nil {
            return fmt.Errorf("bucket 'Search' not found")
        }

        for _, token := range strings.FieldsFunc(query, splitter) {
            wordBucket := searchBucket.Bucket([]byte(token))
            if wordBucket == nil {
                log.Printf("Token '%s' not found, skipping.", token)
                continue
            }

            wordBucket.ForEach(func(infohash, scoreData []byte) error {
                score := bytesToInt(scoreData)
                scoreMap[string(infohash)] += score
                return nil
            })
        }
        return nil
    })

    if err != nil {
        log.Fatal(err)
    }

    // Collect keys and sort them by score in descending order
    keys := make([]string, 0, len(scoreMap))
    for key := range scoreMap {
        keys = append(keys, key)
    }

    sort.SliceStable(keys, func(i, j int) bool {
        return scoreMap[keys[i]] > scoreMap[keys[j]]
    })
    
    // Fetch metadata for each sorted infohash
    for _, key := range keys {
        fmt.Println(ParseMetadata(ShowMetadataForInfohash(key)))
    }
}
