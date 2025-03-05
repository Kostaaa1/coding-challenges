package balancer

import (
	"net/http"
	"sync"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type SW struct {
	servers []*models.Server
	sync.RWMutex
}

func (s *SW) Next(w http.ResponseWriter, r *http.Request) (peer *models.Server) {
	s.Lock()
	defer s.Unlock()

	if len(s.servers) == 0 {
		return nil
	}

	if len(s.servers) == 1 {
		return s.servers[0]
	}

	var total int
	var found bool

	for _, srv := range s.servers {
		if !srv.IsHealthy() {
			continue
		}

		srv.CurrentWeight += srv.Weight
		total += srv.Weight
		if !found || srv.CurrentWeight > peer.CurrentWeight {
			peer = srv
			found = true
		}
	}

	if peer != nil {
		peer.CurrentWeight -= total
	}

	return peer
}

func (s *SW) UpdateServers(servers []*models.Server) {
	s.Lock()
	defer s.Unlock()
	s.servers = servers
}

func NewSmoothWRRStrategy(servers []*models.Server) ILBStrategy {
	return &SW{
		servers: servers,
	}
}
