package strategy

import (
	"net/http"
	"sync"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type SmoothWRR struct {
	servers     []*models.Server
	cw          []int
	totalWeight int
	sync.RWMutex
}

func max(nums []int) (int, int) {
	i, max := 0, nums[0]
	for id, n := range nums {
		if n > max {
			i, max = id, n
		}
	}
	return i, max
}

func (s *SmoothWRR) Next(w http.ResponseWriter, r *http.Request) *models.Server {
	s.Lock()
	defer s.Unlock()

	if len(s.servers) == 0 {
		return nil
	}

	for i, srv := range s.servers {
		s.cw[i] += srv.Weight
	}

	maxId, _ := max(s.cw)
	s.cw[maxId] -= s.totalWeight

	return s.servers[maxId]
}

func NewSmoothWRRStrategy(servers []*models.Server) ILBStrategy {
	cw := make([]int, len(servers))

	totalWeight := 0
	for _, srv := range servers {
		totalWeight += srv.Weight
	}

	return &SmoothWRR{
		servers:     servers,
		cw:          cw,
		totalWeight: totalWeight,
	}
}
