package dht

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/jackpal/bencode-go"
)

var unique = make(map[string]struct{})

// Request structure for get_peers
type GetPeersReq struct {
	T string `bencode:"t"` // Transaction ID
	Y string `bencode:"y"` // Query type (should be 'q')
	Q string `bencode:"q"` // Query method (should be 'get_peers"`
	A struct {
		ID       string `bencode:"id"`        // Your node's ID
		InfoHash string `bencode:"info_hash"` // The target infohash
	} `bencode:"a"`
}

// Response structure for get_peers
type GetPeersResp struct {
	R struct {
		ID     string   `bencode:"id"`     // Queried node's ID
		Token  string   `bencode:"token"`  // Token for announce_peer
		Nodes  string   `bencode:"nodes"`  // Compact node info (optional)
		Values []string `bencode:"values"` // List of peers (optional)
	} `bencode:"r"`
	T string `bencode:"t"` // Transaction ID
	Y string `bencode:"y"` // Response type (should be 'r')
}

// Decode compact peer info (IP:Port)
func decodeCompactPeers(peers []string, infohash string){
	// fmt.Println("Got peers")
	for _, peer := range peers {
		if len(peer) != 6 {
			// fmt.Println("Invalid peer length:", len(peer))
			continue
		}
		ip := net.IP(peer[:4])                            // First 4 bytes are the IP address
		port := binary.BigEndian.Uint16([]byte(peer[4:6])) // Last 2 bytes are the port
		address := fmt.Sprintf("%s:%d", ip, port)
		if _, exists := unique[address]; exists {
			continue
		}
		unique[address] = struct{}{} // Mark as seen
		// fmt.Println(infohash)
		// fmt.Printf("Peer IP: %s, Port: %d\n", ip, port)
		if !CheckInfohashExists(infohash){
			log.Printf("Infohash: %s, Peer IP: %s, Port: %d\n",infohash, ip, port)
			Metadata(address,infohash)
		}
	}
}

// Decode compact node info (NodeID, IP:Port)
func decodeCompactNodes(nodes string,infohash string) {
	const nodeInfoLength = 26 // 20 bytes NodeID + 6 bytes IP:Port
	for i := 0; i+nodeInfoLength <= len(nodes); i += nodeInfoLength {
		// nodeID := nodes[i : i+20]                     // NodeID is 20 bytes
		ip := net.IP(nodes[i+20 : i+24])              // 4 bytes for IPv4 address
		port := binary.BigEndian.Uint16([]byte(nodes[i+24 : i+26])) // 2 bytes for the port
		// fmt.Printf("NodeID: %x, IP: %s, Port: %d\n", nodeID, ip, port)
		address := fmt.Sprintf("%s:%d", ip, port)
		if _, exists := unique[address]; exists {
			continue
		}
		unique[address] = struct{}{}
		getPeer(address, infohash) // Recursively query the node
	}
}

func getPeer(address string, infohash string) {
	// Create the request for get_peers
	req := GetPeersReq{T: "aa", Y: "q", Q: "get_peers"}
	req.A.ID = "abcdefghij0123456789" // Your node's ID

	// Decode the InfoHash from hex string
	infoHashBytes, err := hex.DecodeString(infohash)
	if err != nil {
		// log.Fatalf("Invalid info hash: %v\n", err)
	}
	req.A.InfoHash = string(infoHashBytes)

	var buf bytes.Buffer
	err = bencode.Marshal(&buf, req)
	if err != nil {
		// log.Fatalf("Failed to marshal get_peers request: %v\n", err)
	}

	// Create UDP connection to DHT node
	var d net.Dialer
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := d.DialContext(ctx, "udp4", address)
	if err != nil {
		// fmt.Printf("Failed to dial: %v\n", err)
		return
	}
	defer conn.Close()

	// Send get_peers request
	if _, err := conn.Write(buf.Bytes()); err != nil {
		// fmt.Printf("Failed to send get_peers request: %v\n", err)
		return
	}

	// Read response
	resp := make([]byte, 65535) // Large buffer for response
	conn.SetReadDeadline(time.Now().Add(10 * time.Second)) // 10-second timeout
	n, err := conn.Read(resp)
	if err != nil {
		// fmt.Printf("Failed to read response: %v\n", err)
		return
	}
	resp = resp[:n] // Trim the response to actual size

	// Print raw response for debugging (optional)
	// fmt.Printf("Raw response (hex): %s\n", hex.EncodeToString(resp))

	// Unmarshal response
	var response GetPeersResp
	r := bytes.NewReader(resp)
	err = bencode.Unmarshal(r, &response)
	if err != nil {
		// fmt.Printf("Failed to unmarshal response: %v\n", err)
		return
	}
	// fmt.Println(response.R.Token)
	if len(response.R.Values) > 0 {
		// fmt.Printf("Peers:\n")
		decodeCompactPeers(response.R.Values,infohash)
	} else if response.R.Nodes != "" {
		// fmt.Printf("Nodes:\n")
		decodeCompactNodes(response.R.Nodes,infohash)
	} else {
		// fmt.Println("No peers or nodes found.")
	}
}


func Peers(infohash string) {
	var bootstrapNodes = []string{
		"router.bittorrent.com:6881",
		"dht.transmissionbt.com:6881",
		"router.utorrent.com:6881",
	}
	for _, node := range bootstrapNodes {
		getPeer(node,infohash)
	}
}