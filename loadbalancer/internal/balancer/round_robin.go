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
