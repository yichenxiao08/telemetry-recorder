package workload

import (
	"testing"
	"time"
)

func TestCartValidationAndWork(t *testing.T) {
	if err := (Cart{}).Validate(); err == nil {
		t.Fatal("empty cart accepted")
	}
	bulk := Cart{ID: "bulk", Items: make([]Item, 80)}
	for index := range bulk.Items {
		bulk.Items[index] = Item{SKU: "sku-001", Quantity: 1}
	}
	if err := bulk.Validate(); err != nil {
		t.Fatal(err)
	}
	if CacheHit(bulk) || QueryDelay(bulk) != 405*time.Millisecond {
		t.Fatal("bulk work must occupy a database connection")
	}
	bulk.Items[0].Quantity = 0
	if err := bulk.Validate(); err == nil {
		t.Fatal("invalid quantity accepted")
	}
}
