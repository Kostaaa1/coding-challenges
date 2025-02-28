package strategy

import (
	"net/http"
	"sync/atomic"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type SmoothWRR struct {
	servers     []*models.Server
	cw          []atomic.Int32
	totalWeight int
}

func (s *SmoothWRR) Next(w http.ResponseWriter, r *http.Request) *models.Server {
	if len(s.servers) == 0 {
		return nil
	}

	for i, srv := range s.servers {
		if srv.Healthy {
			s.cw[i].Add(int32(srv.Weight))
		}
	}

	maxIdx := 0
	for i := range s.cw {
		if s.cw[i].Load() > s.cw[maxIdx].Load() {
			maxIdx = i
		}
	}

	s.cw[maxIdx].Add(-int32(s.totalWeight))
	return s.servers[maxIdx]
}

func NewSmoothWRRStrategy(servers []*models.Server) ILBStrategy {
	totalWeight := 0
	for _, srv := range servers {
		totalWeight += srv.Weight
	}

	cw := make([]atomic.Int32, len(servers))

	return &SmoothWRR{
		servers:     servers,
		cw:          cw,
		totalWeight: totalWeight,
	}
}
