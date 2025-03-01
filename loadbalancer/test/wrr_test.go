package test

import (
	"sync"
	"testing"

	"github.com/Kostaaa1/loadbalancer/internal/balancer"
	"github.com/Kostaaa1/loadbalancer/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestWRRConcurrentRequests(t *testing.T) {
	servers := []*models.Server{
		{Name: "server1", Weight: 1, Healthy: true},
		{Name: "server2", Weight: 2, Healthy: true},
		{Name: "server3", Weight: 3, Healthy: true},
	}
	strategy := balancer.NewSmoothWRRStrategy(servers)

	var wg sync.WaitGroup
	concurrency := 100000

	counters := make(map[string]int)
	var counterMutex sync.Mutex

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			srv := strategy.Next(nil, nil)

			if srv == nil {
				t.Errorf("Got nil server, expected a valid server")
				return
			}

			counterMutex.Lock()
			counters[srv.Name]++
			counterMutex.Unlock()
		}()
	}

	wg.Wait()

	server1Count := counters["server1"]
	server2Count := counters["server2"]
	server3Count := counters["server3"]

	t.Logf("Server selections: server1=%d, server2=%d, server3=%d",
		server1Count, server2Count, server3Count)

	assert.GreaterOrEqual(t, server3Count, server2Count)
	assert.GreaterOrEqual(t, server2Count, server1Count)

	totalRequests := server1Count + server2Count + server3Count
	assert.Equal(t, concurrency, totalRequests, "Some requests were not handled")
}
