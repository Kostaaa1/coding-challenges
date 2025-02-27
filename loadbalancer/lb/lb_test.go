package loadbalancer_test

import (
	"sync"
	"testing"

	"github.com/Kostaaa1/loadbalancer/internal/models"
	"github.com/Kostaaa1/loadbalancer/strategy"
	"github.com/stretchr/testify/assert"
)

func TestConcurrentRequests(t *testing.T) {
	// Setup some mock servers
	servers := []*models.Server{
		{Name: "server1", Weight: 1, Healthy: true},
		{Name: "server2", Weight: 2, Healthy: true},
		{Name: "server3", Weight: 3, Healthy: true},
	}

	// Create WRR strategy
	strategy := strategy.NewWRRStrategy(servers)

	// Use WaitGroup to wait for all concurrent requests to finish
	var wg sync.WaitGroup
	concurrency := 1000000 // Number of concurrent requests

	// Create counters for each server
	counters := make(map[string]int)
	var counterMutex sync.Mutex

	// Run the test with multiple concurrent goroutines
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Get next server
			srv := strategy.Next(nil, nil)

			// Check if server is nil
			if srv == nil {
				t.Errorf("Got nil server, expected a valid server")
				return
			}

			// Update counter safely
			counterMutex.Lock()
			counters[srv.Name]++
			counterMutex.Unlock()
		}()
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Get the counts
	server1Count := counters["server1"]
	server2Count := counters["server2"]
	server3Count := counters["server3"]

	// Log the distributions
	t.Logf("Server selections: server1=%d, server2=%d, server3=%d",
		server1Count, server2Count, server3Count)

	// Assert that server selections match weight proportions
	// The exact proportions might vary slightly, but should follow the weight pattern
	assert.GreaterOrEqual(t, server3Count, server2Count)
	assert.GreaterOrEqual(t, server2Count, server1Count)

	// Optionally, check if total requests were handled
	totalRequests := server1Count + server2Count + server3Count
	assert.Equal(t, concurrency, totalRequests, "Some requests were not handled")
}
