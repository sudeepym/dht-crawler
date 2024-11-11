package main

import (
	"dht-crawler/dht"
)

func main() {
	dht.InitDB()
	defer dht.CloseDB()
	dht.CrawlDHT()
}
