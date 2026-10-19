package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/signalscout/signalscout/samples/shop/load"
)

func main() {
	var options load.Options
	flag.StringVar(&options.URL, "url", "http://127.0.0.1:18080/checkout", "Shop checkout URL")
	flag.StringVar(&options.InventoryHealthURL, "inventory-health", "http://127.0.0.1:18082/healthz", "Inventory pool health URL")
	flag.StringVar(&options.Phase, "phase", "manual", "Label in generated request IDs")
	flag.IntVar(&options.Count, "count", 300, "Checkout requests")
	flag.IntVar(&options.Concurrency, "concurrency", 8, "Concurrent HTTP requests")
	flag.Int64Var(&options.Seed, "seed", 1, "Fixed workload seed")
	flag.BoolVar(&options.Bursts, "bursts", true, "Include bulk-cart bursts; requires concurrency >=8")
	flag.Parse()
	result, err := load.Run(context.Background(), options)
	if encodeErr := json.NewEncoder(os.Stdout).Encode(result); encodeErr != nil {
		fmt.Fprintln(os.Stderr, encodeErr)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
