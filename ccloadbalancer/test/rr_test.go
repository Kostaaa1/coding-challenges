package test

import (
	"sync"
	"testing"

	"github.com/Kostaaa1/loadbalancer/internal/models"
	"github.com/Kostaaa1/loadbalancer/internal/strategy"
	"github.com/stretchr/testify/assert"
)

func TestRoundRobinConcurrentRequests(t *testing.T) {
	servers := []*models.Server{
		{Name: "server1", Weight: 1},
		{Name: "server2", Weight: 2},
		{Name: "server3", Weight: 3},
	}
	for _, srv := range servers {
		srv.SetHealthy(true)
	}

	strategy := strategy.NewRoundRobinStrategy(servers)

	var wg sync.WaitGroup
	concurrency := 3000

	counters := make(map[string]int)
	var counterMutex sync.Mutex

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			srv := strategy.Next()
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

func TestServerFailure(t *testing.T) {
	servers := []*models.Server{
		{Name: "server1", Weight: 1},
		{Name: "server2", Weight: 1},
		{Name: "server3", Weight: 1},
	}
	for _, srv := range servers {
		srv.SetHealthy(true)
	}
	strategy := strategy.NewRoundRobinStrategy(servers)

	s1 := strategy.Next()
	s2 := strategy.Next()
	s3 := strategy.Next()
	s4 := strategy.Next()

	assert.Equal(t, "server1", s1.Name)
	assert.Equal(t, "server2", s2.Name)
	assert.Equal(t, "server3", s3.Name)
	assert.Equal(t, "server1", s4.Name)
	servers[1].SetHealthy(false)

	s5 := strategy.Next()
	assert.Equal(t, "server3", s5.Name, "Should skip server2 and return server3")

	s6 := strategy.Next()
	assert.Equal(t, "server1", s6.Name)

	s7 := strategy.Next()
	assert.Equal(t, "server3", s7.Name)

	servers[0].SetHealthy(false)
	servers[2].SetHealthy(false)

	s8 := strategy.Next()
	assert.Nil(t, s8, "Should return nil when no healthy servers")

	servers[1].SetHealthy(true)

	s9 := strategy.Next()
	assert.Equal(t, "server2", s9.Name)

	servers[1].SetHealthy(false)
	servers[2].SetHealthy(true)

	s10 := strategy.Next()
	assert.Equal(t, "server3", s10.Name, "Should find server3 as next healthy server")

	servers[0].SetHealthy(true)
	servers[1].SetHealthy(true)

	s11 := strategy.Next()
	s12 := strategy.Next()
	s13 := strategy.Next()

	assert.Equal(t, "server1", s11.Name)
	assert.Equal(t, "server2", s12.Name)
	assert.Equal(t, "server3", s13.Name)
}
