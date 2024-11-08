package dht

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/jackpal/bencode-go"
)

func Metadata(peerIP, infohash string) {
	conn, err := net.DialTimeout("tcp", peerIP, 10*time.Second)
	if err != nil {
		log.Printf("Failed to connect to peer: %v", err)
		return
	}
	defer conn.Close()

	err = sendStandardHandshake(conn, infohash)
	if err != nil {
		log.Printf("Failed to send handshake: %v", err)
		return
	}

	if err = receivePeerHandshake(conn); err != nil {
		log.Printf("Failed to receive peer handshake: %v", err)
		return
	}

	err = sendExtensionHandshake(conn)
	if err != nil {
		log.Printf("Failed to send extension handshake: %v", err)
		return
	}

	utMetadataID, metadataSize, err := receiveExtensionHandshakeResponse(conn)
	if err != nil {
		log.Printf("Failed to receive extension handshake response: %v", err)
		return
	}

	fmt.Printf("Peer supports ut_metadata with message ID: %d\n", utMetadataID)
	fmt.Printf("Metadata size: %d bytes\n", metadataSize)

	// Retrieve all metadata pieces
	metadata, err := retrieveMetadata(conn, utMetadataID, metadataSize)
	if err != nil {
		log.Printf("Failed to retrieve metadata: %v", err)
		return
	}

	fmt.Println("Metadata retrieved:", string(metadata))
}

func sendStandardHandshake(conn net.Conn, infohash string) error {
	decodedInfohash, err := hex.DecodeString(infohash)
	if err != nil || len(decodedInfohash) != 20 {
		return fmt.Errorf("invalid infohash: %w", err)
	}
	protocol := "BitTorrent protocol"
	reserved := make([]byte, 8)
	reserved[5] |= 0x10
	peerID := "-DE0001-123456789012"

	buf := new(bytes.Buffer)
	buf.WriteByte(byte(len(protocol)))
	buf.WriteString(protocol)
	buf.Write(reserved)
	buf.Write(decodedInfohash)
	buf.WriteString(peerID)

	_, err = conn.Write(buf.Bytes())
	return err
}

func receivePeerHandshake(conn net.Conn) error {
	peerHandshake := make([]byte, 68)
	_, err := conn.Read(peerHandshake)
	if err != nil {
		return err
	}
	if peerHandshake[25]&0x10 == 0 {
		return fmt.Errorf("peer does not support extension protocol")
	}
	return nil
}

func sendExtensionHandshake(conn net.Conn) error {
	extensionHandshake := map[string]interface{}{
		"m": map[string]int{"ut_metadata": 1},
	}
	var handshakeBuffer bytes.Buffer
	err := bencode.Marshal(&handshakeBuffer, extensionHandshake)
	if err != nil {
		return err
	}

	messageLength := uint32(handshakeBuffer.Len() + 2)
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, messageLength)
	buf.WriteByte(20)
	buf.WriteByte(0)

	_, err = conn.Write(append(buf.Bytes(), handshakeBuffer.Bytes()...))
	return err
}


type MetadataResponse struct {
	// Extension messages support
	Extensions map[string]int `bencode:"m"`
	
	// Metadata size
	MetadataSize int `bencode:"metadata_size"`
	
	// Additional fields (optional)
	YourIP     string `bencode:"yourip"`
	ListenPort int    `bencode:"p"`
	Client     string `bencode:"v"`
}

func receiveExtensionHandshakeResponse(conn net.Conn) (int, int, error) {
	resp := make([]byte, 4096)
	_, err := conn.Read(resp)
	if err != nil {
		return 0, 0, err
	}
	resp = resp[3:]
	if resp[1] != 20 || resp[2] != 0 {
		return 0, 0, fmt.Errorf("invalid extension handshake response")
	}
	resp = resp[3:]
	var response MetadataResponse
	err = bencode.Unmarshal(bytes.NewReader(resp), &response)
	if err != nil {
		return 0, 0, err
	}
	fmt.Println(response)
	utMetadataID := 0
	if _, ok := response.Extensions["ut_metadata"]; ok {
		utMetadataID = response.Extensions["ut_metadata"]
	}

	metadataSize := response.MetadataSize

	if utMetadataID == 0 {
		return 0, 0, fmt.Errorf("peer does not support ut_metadata")
	}
	return utMetadataID, metadataSize, nil
}

func retrieveMetadata(conn net.Conn, utMetadataID int, metadataSize int) ([]byte, error) {
	totalPieces := (metadataSize + 16383) / 16384
	var metadata []byte

	for piece := 0; piece < totalPieces; piece++ {
		err := requestMetadataPiece(conn, utMetadataID, piece)
		if err != nil {
			return nil, fmt.Errorf("failed to request piece %d: %w", piece, err)
		}

		data, err := receiveMetadataPiece(conn)
		if err != nil {
			return nil, fmt.Errorf("failed to receive piece %d: %w", piece, err)
		}

		metadata = append(metadata, data...)
	}

	return metadata, nil
}


func requestMetadataPiece(conn net.Conn, utMetadataID int, piece int) error {
	var buf bytes.Buffer

	// Create the request payload
	request := map[string]interface{}{
		"msg_type": 0,  // Requesting a piece
		"piece":    piece,
	}

	// Marshal the request into a buffer
	err := bencode.Marshal(&buf, request)
	if err != nil {
		return err
	}

	// Calculate the message length (including the prefix and ID)
	messageLength := uint32(buf.Len() + 2) // +2 for the extension ID and message ID
	var messageBuffer bytes.Buffer

	// Write the message length to the buffer (BigEndian format)
	binary.Write(&messageBuffer, binary.BigEndian, messageLength)

	// Write the message ID (20 for 'ut_metadata')
	messageBuffer.WriteByte(20) // Message ID for 'ut_metadata'

	// Write the extension ID (this will depend on the peer's handshake)
	messageBuffer.WriteByte(byte(utMetadataID)) // Peer-specific 'ut_metadata' extension ID

	// Append the marshaled request (bencoded dictionary) to the message buffer
	messageBuffer.Write(buf.Bytes())

	// Send the complete message to the connection
	_, err = conn.Write(messageBuffer.Bytes())
	return err
}


type File struct {
	Length	int	`bencode:"length"`
	Path	[]string	`bencode:"path"`
}

type MetaData struct {
	MsgType	int	`bencode:"msg_type"`
	Piece	int	`bencode:"piece"`
	TotalSize	int	`bencode:"total_size"`
	Files	[]File	`bencode:"files"`
	Name	string	`bencode:"name"`
	PieceLength	int	`bencode:"piece length"`
	Pieces	[]byte	`bencode:"pieces"`
}


func receiveMetadataPiece(conn net.Conn) ([]byte, error) {
	response := make([]byte, 655365)
	_, err := conn.Read(response)
	if err != nil {
		return nil, err
	}
	response = response[3:]
	if response[1] != 20 {
		return nil, fmt.Errorf("unexpected message type")
	}
	
	response = response[3:]
	
	fmt.Println(string(response))
	var responseDict MetaData
	err = bencode.Unmarshal(bytes.NewReader(response), &responseDict)
	if err != nil {
		return nil, err
	}

	if responseDict.MsgType != 1 {
		return nil, fmt.Errorf("invalid msg_type in response")
	}
	fmt.Println(responseDict)
	if responseDict.Piece>=0 {
		return []byte(responseDict.Name), nil
	}

	return nil, fmt.Errorf("no piece data found in response")
}
