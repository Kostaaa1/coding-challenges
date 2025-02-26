package strategy

import (
	"net/http"
	"sort"
	"sync"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type WRR struct {
	servers []*models.Server
	cw      []int
	index   int
	sync.RWMutex
}

func (s *WRR) isCycleOver() bool {
	for _, w := range s.cw {
		if w > 0 {
			return false
		}
	}
	return true
}

func (s *WRR) resetWeights() bool {
	for i := range s.cw {
		s.cw[i] = s.servers[i].Weight
	}
	return true
}

func (s *WRR) Next(w http.ResponseWriter, r *http.Request) *models.Server {
	s.Lock()
	defer s.Unlock()

	if len(s.servers) == 0 {
		return nil
	}

	for {
		srv := s.servers[s.index]
		if s.cw[s.index] > 0 && srv.Healthy {
			s.cw[s.index]--
			s.index = (s.index + 1) % len(s.servers)
			return srv
		}

		if s.isCycleOver() {
			s.resetWeights()
		}

		s.index = (s.index + 1) % len(s.servers)
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
