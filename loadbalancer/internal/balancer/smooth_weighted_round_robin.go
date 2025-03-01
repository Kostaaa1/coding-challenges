package balancer

import (
	"net/http"
	"sync"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type SmoothWRR struct {
	servers     []*models.Server
	cw          []int
	totalWeight int
	sync.Mutex
}

func (s *SmoothWRR) Next(w http.ResponseWriter, r *http.Request) *models.Server {
	s.Lock()
	defer s.Unlock()

	if len(s.servers) == 0 {
		return nil
	}

	var selectedSrv *models.Server
	maxWeight := -1
	selectedIdx := -1

	for idx, srv := range s.servers {
		if !srv.Healthy {
			continue
		}

		s.cw[idx] += srv.Weight

		if s.cw[idx] > maxWeight {
			maxWeight = s.cw[idx]
			selectedSrv = srv
			selectedIdx = idx
		}
	}

	if selectedSrv == nil {
		return nil
	}

	s.cw[selectedIdx] -= s.totalWeight
	return selectedSrv
}

func NewSmoothWRRStrategy(servers []*models.Server) ILBStrategy {
	total := 0
	for _, srv := range servers {
		total += srv.Weight
	}
	return &SmoothWRR{
		servers:     servers,
		totalWeight: total,
		cw:          make([]int, len(servers)),
	}
}
