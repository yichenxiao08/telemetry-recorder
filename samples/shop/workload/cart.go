// Package workload defines the shop's deterministic carts and query work.
package workload

import (
	"fmt"
	"hash/fnv"
	"time"
)

type Item struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}
type Cart struct {
	ID    string `json:"id"`
	Items []Item `json:"items"`
}

func (c Cart) Validate() error {
	if c.ID == "" || len(c.ID) > 100 || len(c.Items) == 0 || len(c.Items) > 100 {
		return fmt.Errorf("cart needs an ID and 1..100 items")
	}
	for _, item := range c.Items {
		if item.SKU == "" || len(item.SKU) > 40 || item.Quantity < 1 || item.Quantity > 10 {
			return fmt.Errorf("invalid SKU or quantity")
		}
	}
	return nil
}

// Bulk carts always recheck the database. Ordinary carts use a stable cache
// policy, making cache misses a recurring decoy rather than a failure signal.
func CacheHit(c Cart) bool {
	if len(c.Items) == 0 || len(c.Items) >= 8 {
		return false
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(c.Items[0].SKU))
	return h.Sum32()%10 < 7
}

// QueryDelay models per-line reservation work inside a real Postgres query.
// The same workload is used with both healthy and undersized pools.
func QueryDelay(c Cart) time.Duration { return (5 + time.Duration(len(c.Items))*5) * time.Millisecond }
