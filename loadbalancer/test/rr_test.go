package test

import (
	"sync"
	"testing"

	"github.com/Kostaaa1/loadbalancer/internal/balancer"
	"github.com/Kostaaa1/loadbalancer/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestRRConcurrentRequests(t *testing.T) {
	servers := []*models.Server{
		{Name: "server1", Weight: 1, Healthy: true},
		{Name: "server2", Weight: 2, Healthy: true},
		{Name: "server3", Weight: 3, Healthy: true},
	}

	strategy := balancer.NewRoundRobinStrategy(servers)

	var wg sync.WaitGroup
	concurrency := 3000

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

	assert.Equal(t, server1Count, server2Count)
	assert.Equal(t, server2Count, server3Count)

	totalRequests := server1Count + server2Count + server3Count
	assert.Equal(t, concurrency, totalRequests, "Some requests were not handled")
}

// Server failure
func TestServerFailure(t *testing.T) {
	servers := []*models.Server{
		{Name: "server1", Weight: 1, Healthy: true},
		{Name: "server2", Weight: 1, Healthy: true},
		{Name: "server3", Weight: 1, Healthy: true},
	}
	strategy := balancer.NewRoundRobinStrategy(servers)

	// Test normal round-robin pattern
	s1 := strategy.Next(nil, nil)
	s2 := strategy.Next(nil, nil)
	s3 := strategy.Next(nil, nil)
	s4 := strategy.Next(nil, nil)

	assert.Equal(t, "server1", s1.Name)
	assert.Equal(t, "server2", s2.Name)
	assert.Equal(t, "server3", s3.Name)
	assert.Equal(t, "server1", s4.Name)

	// Mark server2 as unhealthy
	servers[1].Healthy = false

	// Next server in rotation would be server2, but it's unhealthy
	// So we get server3 instead
	s5 := strategy.Next(nil, nil)
	assert.Equal(t, "server3", s5.Name, "Should skip server2 and return server3")

	// Next server would be server3, but we already used it
	// So we get server1
	s6 := strategy.Next(nil, nil)
	assert.Equal(t, "server1", s6.Name)

	// // Next server would be server1, but we already used it
	// // So we cycle back to server3
	s7 := strategy.Next(nil, nil)
	assert.Equal(t, "server3", s7.Name)

	// // Mark all servers as unhealthy
	servers[0].Healthy = false
	servers[2].Healthy = false

	// // Should return nil when no healthy servers
	s8 := strategy.Next(nil, nil)
	assert.Nil(t, s8, "Should return nil when no healthy servers")

	// // Mark server2 as healthy again
	servers[1].Healthy = true

	// // Should select the newly healthy server
	s9 := strategy.Next(nil, nil)
	assert.Equal(t, "server2", s9.Name)

	// // Test cycling through multiple unhealthy servers
	// // Mark server2 unhealthy, server3 healthy
	servers[1].Healthy = false
	servers[2].Healthy = true

	s10 := strategy.Next(nil, nil)
	assert.Equal(t, "server3", s10.Name, "Should find server3 as next healthy server")

	// Make all servers healthy again and verify pattern continues
	servers[0].Healthy = true
	servers[1].Healthy = true

	// The rotation at this point should be:
	s11 := strategy.Next(nil, nil)
	s12 := strategy.Next(nil, nil)
	s13 := strategy.Next(nil, nil)

	// After reaching server3, the next servers should be server1, server2, server3
	assert.Equal(t, "server1", s11.Name)
	assert.Equal(t, "server2", s12.Name)
	assert.Equal(t, "server3", s13.Name)
}
