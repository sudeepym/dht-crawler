package dht

import (
    "encoding/binary"
    "fmt"
    "strings"
    "unicode"
    "sync"
    "github.com/boltdb/bolt"
)

// Constants for score weights and bucket names
const (
    nameTokenWeight     = 20
    fileTokenWeight     = 10
    searchBucketName   = "Search"
    batchSize          = 1000
)

// TokenScorer handles token scoring with configurable weights
type TokenScorer struct {
    nameWeight  int
    fileWeight  int
    cache       *sync.Map
}

// NewTokenScorer creates a new TokenScorer with default weights
func NewTokenScorer() *TokenScorer {
    return &TokenScorer{
        nameWeight:  nameTokenWeight,
        fileWeight:  fileTokenWeight,
        cache:      &sync.Map{},
    }
}

// tokenize splits text into lowercase tokens, cached for performance
func (ts *TokenScorer) tokenize(text string) []string {
    if cached, ok := ts.cache.Load(text); ok {
        return cached.([]string)
    }
    
    tokens := strings.FieldsFunc(strings.ToLower(text), func(c rune) bool {
        return !unicode.IsLetter(c) && !unicode.IsNumber(c)
    })
    
    // Only cache if the text is long enough to be worth caching
    if len(text) > 100 {
        ts.cache.Store(text, tokens)
    }
    
    return tokens
}

// Index indexes a torrent with improved performance and memory usage
func Index(infohash, name string, files []string) error {
    if infohash == "" {
        return fmt.Errorf("empty infohash provided")
    }

    scorer := NewTokenScorer()
    scoreMap := make(map[string]int, 100) // Pre-allocate with reasonable size
    
    // Process name tokens
    if name != "" {
        for _, token := range scorer.tokenize(name) {
            if len(token) > 2 { // Skip very short tokens
                scoreMap[token] += scorer.nameWeight
            }
        }
    }

    // Process file tokens
    for _, file := range files {
        for _, token := range scorer.tokenize(file) {
            if len(token) > 2 { // Skip very short tokens
                scoreMap[token] += scorer.fileWeight
            }
        }
    }

    if len(scoreMap) == 0 {
        return fmt.Errorf("no valid tokens found for indexing infohash: %s", infohash)
    }

    // Batch write to database
    return db.Batch(func(tx *bolt.Tx) error {
        searchBucket, err := tx.CreateBucketIfNotExists([]byte(searchBucketName))
        if err != nil {
            return fmt.Errorf("failed to create search bucket: %v", err)
        }

        // Pre-allocate buffer for score conversion
        scoreBuf := make([]byte, 4)
        
        for token, score := range scoreMap {
            wordBucket, err := searchBucket.CreateBucketIfNotExists([]byte(token))
            if err != nil {
                return fmt.Errorf("failed to create token bucket '%s': %v", token, err)
            }
            
            binary.BigEndian.PutUint32(scoreBuf, uint32(score))
            if err := wordBucket.Put([]byte(infohash), scoreBuf); err != nil {
                return fmt.Errorf("failed to store score for token '%s': %v", token, err)
            }
        }
        return nil
    })
}
