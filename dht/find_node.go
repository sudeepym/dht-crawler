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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
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

// Recursive function to query DHT nodes and expand the search
func CrawlDHT() {
	nodeID := "abcdefghij0123456789"
	target := "mnopqrstuvwxyz123456"
	// Start with bootstrap nodes
	queue := bootstrapNodes

	// Set to track visited nodes
	visited := make(map[string]bool)

	for len(queue) > 0 {
		address := queue[0]
		queue = queue[1:]

		if visited[address] {
			continue
		}
		visited[address] = true

		// fmt.Printf("Querying node: %s\n", address)
		nodes, err := sendFindNodeRequest(target, nodeID, address)
		if err != nil {
			// log.Printf("Error querying %s: %v\n", address, err)
			continue
		}

		// Print and queue new nodes
		for _, node := range nodes {
			if !visited[node] {
				// fmt.Printf("Discovered node: %s\n", node)
				queue = append(queue, node)
			}
		}

		infohashes, err := sendSampleInfohashRequest(nodeID, address, target)
		if err != nil {
			// log.Printf("Error requesting infohashes from %s: %v\n", address, err)
		} else {
			// Print or store the discovered infohashes
			var wg sync.WaitGroup
			for _, hash := range infohashes {
				// fmt.Printf("Discovered infohash: %s\n", hash)
				if !CheckInfohashExists(hash) {
					wg.Add(1)
					go func(ih string) {
						defer wg.Done()
						Peers(ih)
					}(hash)
				}
			}
			wg.Wait()
		}
	}
}
