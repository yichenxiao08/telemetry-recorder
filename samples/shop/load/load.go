// Package load sends reproducible cart variation and bursts to the shop sample.
package load

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/signalscout/signalscout/samples/shop/workload"
)

type Options struct {
	URL, InventoryHealthURL, Phase string
	Count, Concurrency             int
	Seed                           int64
	Bursts                         bool
}
type Record struct {
	RequestID string `json:"request_id"`
	TraceID   string `json:"trace_id"`
	Status    int    `json:"status"`
	Lines     int    `json:"lines"`
	CacheHit  bool   `json:"cache_hit"`
	Error     string `json:"error,omitempty"`
}
type Result struct {
	Phase         string   `json:"phase"`
	Seed          int64    `json:"seed"`
	Requests      int      `json:"requests"`
	Passing       int      `json:"passing"`
	Failing       int      `json:"failing"`
	NetworkErrors int      `json:"network_errors"`
	Seconds       float64  `json:"seconds"`
	Records       []Record `json:"records"`
}

func cart(random *rand.Rand, id string, lines int, cacheMiss bool) workload.Cart {
	if lines == 0 {
		lines = 1 + random.Intn(5)
	}
	c := workload.Cart{ID: id}
	for index := 0; index < lines; index++ {
		c.Items = append(c.Items, workload.Item{SKU: fmt.Sprintf("sku-%03d", random.Intn(1000)), Quantity: 1 + random.Intn(3)})
	}
	for cacheMiss && workload.CacheHit(c) {
		c.Items[0].SKU = fmt.Sprintf("sku-%03d", random.Intn(1000))
	}
	return c
}

func request(ctx context.Context, client *http.Client, url string, c workload.Cart) Record {
	result := Record{RequestID: c.ID, Lines: len(c.Items), CacheHit: workload.CacheHit(c)}
	data, err := json.Marshal(c)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		result.Error = err.Error()
		return result
	}
	r.Header.Set("Content-Type", "application/json")
	response, err := client.Do(r)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	_, err = io.Copy(io.Discard, response.Body)
	closeErr := response.Body.Close()
	result.Status = response.StatusCode
	result.TraceID = response.Header.Get("X-Trace-ID")
	if err != nil {
		result.Error = err.Error()
	} else if closeErr != nil {
		result.Error = closeErr.Error()
	}
	return result
}

func waitBusy(ctx context.Context, client *http.Client, url string) error {
	deadline, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	for {
		r, err := http.NewRequestWithContext(deadline, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("pool health request: %w", err)
		}
		response, err := client.Do(r)
		if err == nil {
			var health struct {
				InUse int32 `json:"in_use"`
			}
			err = json.NewDecoder(response.Body).Decode(&health)
			_ = response.Body.Close()
			if err == nil && response.StatusCode == http.StatusOK && health.InUse >= 5 {
				return nil
			}
		}
		select {
		case <-deadline.Done():
			return fmt.Errorf("could not establish five busy pool connections before the bulk-query window ended: %w", deadline.Err())
		case <-time.After(2 * time.Millisecond):
		}
	}
}

func Run(ctx context.Context, options Options) (Result, error) {
	result := Result{Phase: options.Phase, Seed: options.Seed}
	if options.Count < 1 || options.Concurrency < 1 {
		return result, fmt.Errorf("count and concurrency must be positive")
	}
	if options.Bursts && options.Count >= 8 && (options.Concurrency < 8 || options.InventoryHealthURL == "") {
		return result, fmt.Errorf("burst load requires concurrency >=8 and inventory health URL")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = options.Concurrency
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	random := rand.New(rand.NewSource(options.Seed))
	records := make(chan Record, options.Count)
	limit := make(chan struct{}, options.Concurrency)
	started := time.Now()
	var workers sync.WaitGroup
	submit := func(c workload.Cart) {
		workers.Add(1)
		go func() {
			defer workers.Done()
			select {
			case limit <- struct{}{}:
				defer func() { <-limit }()
				records <- request(ctx, client, options.URL, c)
			case <-ctx.Done():
				records <- Record{RequestID: c.ID, Error: ctx.Err().Error()}
			}
		}()
	}
	// Workload randomness is generated before dispatch. The semaphore bounds HTTP
	// concurrency; records is sized for all requests, so workers never block on it.
	for start := 0; start < options.Count; start += 100 {
		end := start + 100
		if end > options.Count {
			end = options.Count
		}
		next := start
		if options.Bursts && end-start >= 8 {
			for ; next < start+5; next++ {
				submit(cart(random, fmt.Sprintf("%s-%d-%04d", options.Phase, options.Seed, next), 80, true))
			}
			if err := waitBusy(ctx, client, options.InventoryHealthURL); err != nil {
				cancel()
				workers.Wait()
				return result, err
			}
			for ; next < start+8; next++ {
				submit(cart(random, fmt.Sprintf("%s-%d-%04d", options.Phase, options.Seed, next), 0, true))
			}
			workers.Wait()
		}
		for ; next < end; next++ {
			submit(cart(random, fmt.Sprintf("%s-%d-%04d", options.Phase, options.Seed, next), 0, false))
		}
		workers.Wait()
	}
	result.Seconds = time.Since(started).Seconds()
	close(records)
	for record := range records {
		result.Records = append(result.Records, record)
		result.Requests++
		if record.Error != "" {
			result.NetworkErrors++
		} else if record.Status >= 200 && record.Status < 300 {
			result.Passing++
		} else if record.Status >= 500 {
			result.Failing++
		} else {
			return result, fmt.Errorf("unexpected shop HTTP status %d", record.Status)
		}
	}
	sort.Slice(result.Records, func(i, j int) bool { return result.Records[i].RequestID < result.Records[j].RequestID })
	if result.NetworkErrors > 0 {
		return result, fmt.Errorf("%d load requests had transport errors", result.NetworkErrors)
	}
	return result, nil
}
