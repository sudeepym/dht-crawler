package dht

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"time"

	"github.com/jackpal/bencode-go"
)

type SampleInfohashReq struct {
	T string `bencode:"t"`
	Y string `bencode:"y"`
	Q string `bencode:"q"`
	A struct {
		ID     string `bencode:"id"`
		Target string `bencode:"target"`
	} `bencode:"a"`
}

type SampleInfohashResp struct {
	R struct {
		ID       string `bencode:"id"`      // 20-byte peer ID
		Interval int    `bencode:"interval"` // int for interval
		Nodes    string `bencode:"nodes"`   // Binary compact node info (string)
		Num      int    `bencode:"num"`     // int for number of nodes/samples
		Samples  string `bencode:"samples"` // Binary compact sample info (string)
	} `bencode:"r"`
	T string `bencode:"t"`
	Y string `bencode:"y"`
}

func sendSampleInfohashRequest(nodeID, address, target string) ([]string, error) {
	// Create the request
	req := SampleInfohashReq{T: "aa", Y: "q", Q: "sample_infohashes"}
	req.A.ID = nodeID
	req.A.Target = target

	var buf bytes.Buffer
	err := bencode.Marshal(&buf, req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sample_infohashes request: %v", err)
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
		return nil, fmt.Errorf("failed to send sample_infohashes request: %v", err)
	}

	// Read response
	resp := make([]byte, 65535)
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	n, err := conn.Read(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}
	resp = resp[:n] // Trim the response to actual size
	// fmt.Printf("Response from %s: %x\n", address, resp) // Print raw response in hex format

	// Unmarshal response
	var response SampleInfohashResp
	r := bytes.NewReader(resp)
	err = bencode.Unmarshal(r, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	// Parse and return the infohashes (if available)
	infohashes := parseInfohashes([]byte(response.R.Samples)) // Assuming it's compact node format
	return infohashes, nil
}

// Function to parse infohashes from the samples field
func parseInfohashes(samples []byte) []string {
	var infohashes []string
	if len(samples)%20 != 0 {
		fmt.Println("Warning: samples length is not a multiple of 20 bytes")
	}
	for i := 0; i+20 <= len(samples); i += 20 {
		infohash := fmt.Sprintf("%x", samples[i:i+20]) // Convert each 20-byte hash to a hex string
		infohashes = append(infohashes, infohash)
	}
	return infohashes
}
