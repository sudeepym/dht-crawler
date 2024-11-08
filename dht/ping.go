package dht

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/jackpal/bencode-go"
)

type DHTPingQuery struct{
	T string	`bencode:"t"`
	Y string	`bencode:"y"`
	Q string	`bencode:"q"`
	A struct {
		ID string	`bencode:"id"`
	}	`bencode:"a"`
}

type DHTPingResp struct{
	IP string  `bencode:"ip"`  // Assuming this is a string or could be []byte if binary
    R  struct {
        ID string `bencode:"id"` // 20-byte peer ID
    } `bencode:"r"`
    T string `bencode:"t"`
    Y string `bencode:"y"`
}

func Ping(){
	req := DHTPingQuery{T:"aa",Y:"q",Q:"ping"}
	req.A.ID="asdfghjklm1234567890"
	var buf bytes.Buffer
	err := bencode.Marshal(&buf,req)
	if err!=nil{
		panic(err)
	}
	fmt.Println(buf.String())
	var d net.Dialer
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, err := d.DialContext(ctx, "udp4", "router.bittorrent.com:6881")
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Write(buf.Bytes()); err != nil {
		log.Fatal(err)
	}
	resp := make([]byte,100)
	_,err=conn.Read(resp)
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}
	fmt.Println(string(resp))
	var response DHTPingResp
	r:= bytes.NewReader(resp)
	err = bencode.Unmarshal(r,&response)
	if err!=nil{
		panic(err)
	}
}