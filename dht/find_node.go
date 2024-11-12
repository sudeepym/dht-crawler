package dht

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	// "log"
	"net"
	"time"

	"github.com/jackpal/bencode-go"
)

// Structs for the KRPC find_node request and response
type FindNodeReq struct {
	T string `bencode:"t"`
	Y string `bencode:"y"`
	Q string `bencode:"q"`
	A struct {
		ID     string `bencode:"id"`
		Target string `bencode:"target"`
	} `bencode:"a"`
}

type FindNodeResp struct {
	IP string `bencode:"ip"`
	R  struct {
		ID    string `bencode:"id"`
		Nodes string `bencode:"nodes"` // Compact node info
	} `bencode:"r"`
	T string `bencode:"t"`
	Y string `bencode:"y"`
}

// Bootstrap DHT nodes
var bootstrapNodes = []string{
	"router.bittorrent.com:6881",
	"router.utorrent.com:6881",
	"dht.transmissionbt.com:6881",
}

// Function to parse the compact node info from the response
func parseCompactNodes(compact string) []string {
	var nodes []string
	for i := 0; i < len(compact); i += 26 {
		node := compact[i : i+26]
		ip := net.IP(node[20:24])
		port := int(node[24])<<8 | int(node[25])
		nodes = append(nodes, fmt.Sprintf("%s:%d", ip.String(), port))
	}
	return nodes
}

// Function to send the find_node request to a DHT node
func sendFindNodeRequest(target, nodeID, address string) ([]string, error) {
	// Create the request
	req := FindNodeReq{T: "aa", Y: "q", Q: "find_node"}
	req.A.ID = nodeID
	req.A.Target = target
	var buf bytes.Buffer
	err := bencode.Marshal(&buf, req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create UDP connection
	var d net.Dialer
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()
	conn, err := d.DialContext(ctx, "udp4", address)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %v", err)
	}
	defer conn.Close()

	// Send request
	if _, err := conn.Write(buf.Bytes()); err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	// Read response
	resp := make([]byte, 65535)
	conn.SetReadDeadline(time.Now().Add(requestTimeout))
	_, err = conn.Read(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Unmarshal response
	var response FindNodeResp
	r := bytes.NewReader(resp)
	err = bencode.Unmarshal(r, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	// Parse and return node list
	nodes := parseCompactNodes(response.R.Nodes)
	return nodes, nil
}

// Configuration constants
const (
	maxConcurrentConnections = 100
	connectionTimeout       = 5 * time.Second
	requestTimeout         = 10 * time.Second
	maxQueueSize          = 10000
	cleanupInterval       = 5 * time.Minute
)

// Global connection limiter
var (
	semaphore    = make(chan struct{}, maxConcurrentConnections)
	activeNodes  = sync.Map{}
	// lastCleanup  = time.Now()
)

// NodeInfo stores information about each DHT node
type NodeInfo struct {
	lastAccessed time.Time
	failures     int
}

// Improved CrawlDHT with connection pooling and rate limiting
func CrawlDHT() {
	nodeID := "abcdefghij0123456789"
	target := "mnopqrstuvwxyz123456"
	
	// Use bounded queue for nodes
	queue := make(chan string, maxQueueSize)
	for _, node := range bootstrapNodes {
		queue <- node
	}

	// Start cleanup goroutine
	go periodicCleanup()

	var wg sync.WaitGroup
	for i := 0; i < maxConcurrentConnections; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for address := range queue {
				processNode(address, nodeID, target, queue)
			}
		}()
	}

	wg.Wait()
}

// Process a single node with proper error handling and backoff
func processNode(address, nodeID, target string, queue chan string) {
	// Acquire semaphore
	semaphore <- struct{}{}
	defer func() { <-semaphore }()

	// Check node health
	if !isNodeHealthy(address) {
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()

	// Try to establish connection
	conn, err := createConnection(ctx, address)
	if err != nil {
		markNodeFailure(address)
		return
	}
	defer conn.Close()

	// Process find_node request
	nodes, err := sendFindNodeRequestWithTimeout(conn, target, nodeID)
	if err != nil {
		markNodeFailure(address)
		return
	}

	// Queue new nodes with bounds checking
	for _, node := range nodes {
		select {
		case queue <- node:
			// Node added to queue
		default:
			// Queue is full, skip this node
			// log.Printf("Queue is full, skipping node: %s", node)
		}
	}

	// Process infohashes with bounded concurrency
	processInfohashes(address, nodeID, target)
}

// Check if a node is healthy enough to process
func isNodeHealthy(address string) bool {
	if value, ok := activeNodes.Load(address); ok {
		nodeInfo := value.(NodeInfo)
		if nodeInfo.failures > 3 {
			return false
		}
		if time.Since(nodeInfo.lastAccessed) < time.Minute {
			return false
		}
	}
	return true
}

// Mark node failure and update its status
func markNodeFailure(address string) {
	if value, ok := activeNodes.Load(address); ok {
		nodeInfo := value.(NodeInfo)
		nodeInfo.failures++
		activeNodes.Store(address, nodeInfo)
	} else {
		activeNodes.Store(address, NodeInfo{
			lastAccessed: time.Now(),
			failures:     1,
		})
	}
}

// Create connection with timeout
func createConnection(ctx context.Context, address string) (net.Conn, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "udp4", address)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %v", err)
	}
	return conn, nil
}

// Process infohashes with proper concurrency control
func processInfohashes(address, nodeID, target string) {
	infohashes, err := sendSampleInfohashRequest(nodeID, address, target)
	if err != nil {
		return
	}

	// Use bounded worker pool for processing infohashes
	workerPool := make(chan struct{}, 20)
	var wg sync.WaitGroup

	for _, hash := range infohashes {
		if !CheckInfohashExists(hash) {
			wg.Add(1)
			workerPool <- struct{}{} // Acquire worker
			
			go func(ih string) {
				defer wg.Done()
				defer func() { <-workerPool }() // Release worker
				
				ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
				defer cancel()
				
				done := make(chan struct{})
				go func() {
					Peers(ih)
					close(done)
				}()

				select {
				case <-ctx.Done():
					// log.Printf("Timeout processing infohash: %s", ih)
				case <-done:
					// Successfully processed
				}
			}(hash)
		}
	}

	wg.Wait()
}

// Periodic cleanup of inactive nodes
func periodicCleanup() {
	ticker := time.NewTicker(cleanupInterval)
	for range ticker.C {
		now := time.Now()
		activeNodes.Range(func(key, value interface{}) bool {
			nodeInfo := value.(NodeInfo)
			if now.Sub(nodeInfo.lastAccessed) > cleanupInterval {
				activeNodes.Delete(key)
			}
			return true
		})
	}
}

// Additional helper function to enforce timeouts on find_node requests
func sendFindNodeRequestWithTimeout(conn net.Conn, target, nodeID string) ([]string, error) {
	done := make(chan []string, 1)
	errChan := make(chan error, 1)

	go func() {
		nodes, err := sendFindNodeRequest(target, nodeID, conn.RemoteAddr().String())
		if err != nil {
			errChan <- err
			return
		}
		done <- nodes
	}()

	select {
	case nodes := <-done:
		return nodes, nil
	case err := <-errChan:
		return nil, err
	case <-time.After(requestTimeout):
		return nil, fmt.Errorf("request timed out")
	}
}
