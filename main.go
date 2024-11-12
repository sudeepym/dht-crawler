package main

import (
	"context"
	"dht-crawler/dht"
	"time"
)

func main() {
	dht.InitDB()
	defer dht.CloseDB()
	ctx, cancel := context.WithCancel(context.Background())
    go func() {
        // Simulate stopping after a duration, or use a condition to trigger cancel
        time.Sleep(60 * time.Second)
        cancel()
    }()
    dht.CrawlDHT(ctx)
}
