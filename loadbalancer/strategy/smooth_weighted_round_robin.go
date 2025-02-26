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

func maxIndex(nums []int) int {
	maxId := 0
	for i := 1; i < len(nums); i++ {
		if nums[i] > nums[maxId] {
			maxId = i
		}
	}
	return maxId
}

func (s *SmoothWRR) Next(w http.ResponseWriter, r *http.Request) *models.Server {
	s.Lock()
	defer s.Unlock()

	if len(s.servers) == 0 {
		return nil
	}

	for i, srv := range s.servers {
		if srv.Healthy {
			s.cw[i] += srv.Weight
		}
	}

	maxIdx := maxIndex(s.cw)
	s.cw[maxIdx] -= s.totalWeight

	return s.servers[maxIdx]
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
