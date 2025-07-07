# DHT BitTorrent Client

A high-performance DHT (Distributed Hash Table) BitTorrent client written in Go that crawls the BitTorrent network to discover, index, and search torrent metadata.

## Features

- **DHT Network Crawling**: Discovers peers and torrents across the BitTorrent DHT network
- **Metadata Extraction**: Retrieves torrent metadata including names and file lists
- **Full-Text Search**: Indexes and searches torrents by name and file content
- **Peer Discovery**: Finds peers for specific torrents using DHT queries
- **Database Storage**: Persistent storage using BoltDB for metadata and search indices
- **Concurrent Processing**: High-performance concurrent crawling and metadata retrieval

## Architecture

The project consists of several key components:

### Core Modules

- **`metadata.go`**: Handles BitTorrent protocol handshakes and metadata retrieval
- **`find_node.go`**: Implements DHT find_node queries for network discovery
- **`get_peer.go`**: Handles DHT get_peers queries for peer discovery
- **`sample_infohashes.go`**: Samples infohashes from DHT nodes
- **`index.go`**: Creates searchable indices from torrent metadata
- **`query.go`**: Performs full-text search across indexed torrents
- **`bolt.go`**: Database operations for metadata storage
- **`bucket.go`**: Database bucket management and debugging utilities

## Installation

### Prerequisites

- Go 1.16 or higher
- Git

### Dependencies

```bash
go get github.com/boltdb/bolt
go get github.com/jackpal/bencode-go
```

### Setup

1. Clone the repository:
```bash
git clone <repository-url>
cd dht-client
```

2. Install dependencies:
```bash
go mod tidy
```

3. Build the project:
```bash
go build
```

## Usage

### Initialize Database

Before using the client, initialize the database:

```go
import "path/to/dht"

func main() {
    // Initialize database connection
    if err := dht.InitDB(); err != nil {
        log.Fatal("Failed to initialize database:", err)
    }
    defer dht.CloseDB()
    
    // Your code here
}
```

### Basic Operations

#### 1. Crawl DHT Network

```go
import (
    "context"
    "time"
)

func crawlNetwork() {
    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
    defer cancel()
    
    // Start crawling
    dht.CrawlDHT(ctx)
}
```

#### 2. Search Torrents

```go
func searchTorrents(query string) {
    results, err := dht.Query(query)
    if err != nil {
        log.Fatal("Search failed:", err)
    }
    
    for _, result := range results {
        fmt.Printf("Name: %s\n", result.Name)
        fmt.Printf("Infohash: %s\n", result.Infohash)
        fmt.Printf("Files: %v\n", result.Files)
        fmt.Println("---")
    }
}
```

#### 3. Get Peers for Specific Torrent

```go
func findPeers(infohash string) {
    dht.Peers(infohash)
}
```

#### 4. Check if Torrent Exists

```go
func checkTorrent(infohash string) {
    exists := dht.CheckInfohashExists(infohash)
    if exists {
        fmt.Println("Torrent found in database")
        metadata := dht.ShowMetadataForInfohash(infohash)
        name, files := dht.ParseMetadata(metadata)
        fmt.Printf("Name: %s\n", name)
        fmt.Printf("Files: %v\n", files)
    }
}
```

#### 5. List All Torrents

```go
func listTorrents() {
    infohashes := dht.ShowInfohashes()
    fmt.Printf("Found %d torrents\n", len(infohashes))
    for i, hash := range infohashes {
        fmt.Printf("%d: %s\n", i+1, hash)
    }
}
```

### Advanced Usage

#### Custom Indexing

```go
func indexTorrent(infohash, name string, files []string) {
    err := dht.Index(infohash, name, files)
    if err != nil {
        log.Printf("Failed to index torrent: %v", err)
    }
}
```

#### Database Management

```go
// Delete a torrent
func deleteTorrent(infohash string) {
    err := dht.DeleteInfohash(infohash)
    if err != nil {
        log.Printf("Failed to delete torrent: %v", err)
    }
}

// Check indexing status
func checkIndexing() {
    dht.CheckIndexing()
}
```

## Configuration

### Performance Tuning

The client includes several configurable parameters in `find_node.go`:

```go
const (
    maxConcurrentConnections = 100        // Maximum concurrent DHT connections
    connectionTimeout       = 5 * time.Second   // Connection timeout
    requestTimeout         = 10 * time.Second   // Request timeout
    maxQueueSize          = 10000               // Maximum queue size for nodes
    cleanupInterval       = 5 * time.Minute     // Cleanup interval for inactive nodes
)
```

### Search Configuration

Search scoring weights can be adjusted in `index.go`:

```go
const (
    nameTokenWeight     = 20    // Weight for tokens found in torrent names
    fileTokenWeight     = 10    // Weight for tokens found in file names
    batchSize          = 1000   // Database batch size for indexing
)
```

## Database Schema

The client uses BoltDB with the following buckets:

- **`Metadata`**: Stores raw torrent metadata keyed by infohash
- **`Search`**: Contains inverted index for full-text search
  - Sub-buckets for each search token
  - Maps infohash to relevance score

## Protocol Support

- **DHT Protocol**: Implements core DHT operations (find_node, get_peers, sample_infohashes)
- **BitTorrent Protocol**: Supports handshake and extension protocol for metadata retrieval
- **Extension Protocol**: Implements ut_metadata extension for metadata exchange

## Performance Characteristics

- **Concurrent Processing**: Handles up to 100 concurrent connections
- **Memory Efficient**: Uses streaming for large metadata files
- **Scalable Indexing**: Batch operations for database writes
- **Connection Pooling**: Reuses connections where possible
- **Rate Limiting**: Prevents overwhelming DHT nodes

## Error Handling

The client includes comprehensive error handling:

- Connection timeouts and retries
- Invalid metadata detection
- Database transaction rollbacks
- Graceful degradation on peer failures

## Security Considerations

- **Network Security**: Only connects to public DHT nodes
- **Data Validation**: Validates all received metadata
- **Resource Limits**: Prevents memory exhaustion attacks
- **Connection Limits**: Prevents connection flooding

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Disclaimer

This software is intended for educational and research purposes. Users are responsible for complying with local laws and regulations regarding BitTorrent usage. The authors are not responsible for any misuse of this software.

## Troubleshooting

### Common Issues

1. **Database locked**: Ensure only one instance is running
2. **Connection timeouts**: Check network connectivity and firewall settings
3. **High memory usage**: Reduce `maxConcurrentConnections` if needed
4. **Slow indexing**: Increase `batchSize` for better performance

### Debug Mode

Enable debug output by uncommenting log statements in the source code.

## Support

For questions and support, please open an issue on the project repository.
