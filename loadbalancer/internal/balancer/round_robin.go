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
}

func (s *RoundRobin) UpdateServers(servers []*models.Server) {
	s.Lock()
	defer s.Unlock()
	s.servers = servers
}

func (s *RoundRobin) Next(w http.ResponseWriter, r *http.Request) *models.Server {
	if len(s.servers) == 0 {
		return nil
	}

	s.Lock()
	defer s.Unlock()

	for i := 1; i < len(s.servers); i++ {
		c := (s.index + i) % len(s.servers)
		srv := s.servers[c]
		if srv.IsHealthy() {
			s.index = c
			return srv
		}
	}

	return nil
}

func NewRoundRobinStrategy(servers []*models.Server) ILBStrategy {
	return &RoundRobin{
		servers: servers,
		index:   -1,
	}
}

// atomic - uneven distribution
// type RoundRobin struct {
// 	servers []*models.Server
// 	index   atomic.Int32
// }
// func NewRoundRobinStrategy(servers []*models.Server) ILBStrategy {
// 	rr := &RoundRobin{
// 		servers: servers,
// 	}
// 	rr.index.Store(0)
// 	return rr
// }
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
// 		current := s.index.Load()
// 		next := (current + 1) % serverCount
// 		s.index.CompareAndSwap(current, next)
// 		srv := s.servers[current]
// 		if srv.Healthy {
// 			return srv
// 		}
// 	}
// }
