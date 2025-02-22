package strategy

import (
	"net/http"
	"sync"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type RoundRobinStrategy struct {
	index   int
	servers []*models.Server
	sync.RWMutex
}

func NewRoundRobinStrategy() ILBStrategy {
	return &RoundRobinStrategy{index: 0}
}

func (s *RoundRobinStrategy) Next(w http.ResponseWriter, r *http.Request) *models.Server {
	s.Lock()
	defer s.Unlock()

	if len(s.servers) == 0 {
		return nil
	}

	srv := s.servers[s.index]
	s.index = (s.index + 1) % len(s.servers)
	return srv
}

func (s *RoundRobinStrategy) UpdateServers(servers []*models.Server) {
	s.Lock()
	defer s.Unlock()
	s.servers = servers
	s.index = 0
}
