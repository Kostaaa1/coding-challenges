package balancer

import (
	"net/http"
	"sort"
	"sync"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type WRR struct {
	servers []*models.Server
	index   int
	cw      []int
	sync.RWMutex
}

func (s *WRR) Next(w http.ResponseWriter, r *http.Request) *models.Server {
	s.Lock()
	defer s.Unlock()

	if len(s.servers) == 0 {
		return nil
	}

	if s.areWeightsZero() {
		s.resetWeights()
	}

	for i := 0; i < len(s.servers); i++ {
		current := s.index
		s.index = (s.index + 1) % len(s.servers)

		srv := s.servers[current]
		if srv.IsHealthy() && s.cw[current] > 0 {
			s.cw[current]--
			return srv
		}
	}

	if s.areWeightsZero() {
		s.resetWeights()
		return s.Next(w, r)
	}

	return nil
}

func (s *WRR) resetWeights() {
	for i, srv := range s.servers {
		s.cw[i] = srv.Weight
	}
}

func (s *WRR) areWeightsZero() bool {
	for i, srv := range s.servers {
		if srv.IsHealthy() && s.cw[i] > 0 {
			return false
		}
	}
	return true
}

func (s *WRR) UpdateServers(servers []*models.Server) {
	s.Lock()
	defer s.Unlock()
	s.servers = servers
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Weight > servers[j].Weight
	})
	for i, srv := range servers {
		s.cw[i] = srv.Weight
	}
}

func NewWRRStrategy(servers []*models.Server) ILBStrategy {
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Weight > servers[j].Weight
	})
	cw := make([]int, len(servers))
	for i, srv := range servers {
		cw[i] = srv.Weight
	}
	return &WRR{
		servers: servers,
		cw:      cw,
		index:   0,
	}
}
