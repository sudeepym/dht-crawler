package dht

import (
	"encoding/binary"
	"fmt"
	"sort"

	"github.com/boltdb/bolt"
)

type SearchResult struct {
	Infohash string
	Name     string
	Files    []string
}

// QueryResult represents a search result with its score
type QueryResult struct {
    SearchResult
    Score int
}

// Query performs an optimized search query
func Query(query string) ([]SearchResult, error) {
    if query == "" {
        return nil, fmt.Errorf("empty query provided")
    }

    scorer := NewTokenScorer()
    tokens := scorer.tokenize(query)
    if len(tokens) == 0 {
        return nil, fmt.Errorf("no valid tokens in query")
    }

    // Use channels for concurrent processing
    scoresChan := make(chan map[string]int, 1)
    errorsChan := make(chan error, 1)

    // Process scores concurrently
    go func() {
        scoreMap := make(map[string]int)
        
        err := db.View(func(tx *bolt.Tx) error {
            searchBucket := tx.Bucket([]byte(searchBucketName))
            if searchBucket == nil {
                return fmt.Errorf("search bucket not found")
            }

            for _, token := range tokens {
                if len(token) <= 2 {
                    continue
                }

                wordBucket := searchBucket.Bucket([]byte(token))
                if wordBucket == nil {
                    continue
                }

                if err := wordBucket.ForEach(func(infohash, scoreData []byte) error {
                    score := int(binary.BigEndian.Uint32(scoreData))
                    scoreMap[string(infohash)] += score
                    return nil
                }); err != nil {
                    return err
                }
            }
            return nil
        })

        if err != nil {
            errorsChan <- err
            return
        }
        scoresChan <- scoreMap
    }()

    // Wait for results
    select {
    case err := <-errorsChan:
        return nil, err
    case scoreMap := <-scoresChan:
        if len(scoreMap) == 0 {
            return []SearchResult{}, nil
        }

        // Create sorted results
        results := make([]QueryResult, 0, len(scoreMap))
        for infohash, score := range scoreMap {
            name, files := ParseMetadata(ShowMetadataForInfohash(infohash))
            results = append(results, QueryResult{
                SearchResult: SearchResult{
                    Infohash: infohash,
                    Name:     name,
                    Files:    files,
                },
                Score: score,
            })
        }

        // Sort results by score
        sort.Slice(results, func(i, j int) bool {
            return results[i].Score > results[j].Score
        })

        // Convert to final format
        finalResults := make([]SearchResult, len(results))
        for i, r := range results {
            finalResults[i] = r.SearchResult
        }

        return finalResults, nil
    }
}