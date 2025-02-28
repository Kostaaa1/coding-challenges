package balancer

import (
	"net/http"
	"sync"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type RoundRobin struct {
	servers []*models.Server
	index   int
	sync.Mutex
	// index   atomic.Int32
}

func NewRoundRobinStrategy(servers []*models.Server) *RoundRobin {
	rr := &RoundRobin{
		servers: servers,
	}
	// rr.index.Store(0)
	return rr
}

// func (s *RoundRobin) Next(w http.ResponseWriter, r *http.Request) *models.Server {
// 	if len(s.servers) == 0 {
// 		return nil
// 	}

// 	serverCount := int32(len(s.servers))
// 	iterator := int32(0)

// 	for {
// 		if iterator >= serverCount {
// 			return nil
// 		}
// 		iterator++

// 		// Get the current index atomically
// 		currentIndex := s.index.Load()

// 		// Increment the index atomically and wrap around using modulo
// 		// The CompareAndSwap ensures the operation is atomic even during concurrent access
// 		nextIndex := (currentIndex + 1) % serverCount
// 		s.index.CompareAndSwap(currentIndex, nextIndex)

// 		// Get the server at the current index
// 		srv := s.servers[currentIndex]

// 		if srv.Healthy {
// 			return srv
// 		}
// 	}
// }

func (s *RoundRobin) Next(w http.ResponseWriter, r *http.Request) *models.Server {
	if len(s.servers) == 0 {
		return nil
	}

	s.Lock()
	defer s.Unlock()

	iterator := 0

	for {
		if iterator == len(s.servers) {
			return nil
		}
		iterator++

		srv := s.servers[s.index]
		s.index = (s.index + 1) % len(s.servers)
		if srv.Healthy {
			return srv
		}
	}
}
