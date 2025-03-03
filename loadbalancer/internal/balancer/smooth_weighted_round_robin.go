package balancer

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type SmoothWRR struct {
	servers []*models.Server
	cw      []int
	sync.Mutex
}

func (s *SmoothWRR) Next(w http.ResponseWriter, r *http.Request) *models.Server {
	s.Lock()
	defer s.Unlock()

	if len(s.servers) == 0 {
		return nil
	}

	healthyTotal := 0
	for _, srv := range s.servers {
		if srv.IsHealthy() {
			healthyTotal += srv.Weight
		}
	}
	if healthyTotal == 0 {
		return nil
	}

	var selectedSrv *models.Server
	maxWeight := -1
	selectedIdx := -1

	for idx, srv := range s.servers {
		if !srv.IsHealthy() {
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
		fmt.Println("no selected")
		return nil
	}

	s.cw[selectedIdx] -= healthyTotal
	return selectedSrv
}

func NewSmoothWRRStrategy(servers []*models.Server) ILBStrategy {
	cw := make([]int, len(servers))
	return &SmoothWRR{
		servers: servers,
		cw:      cw,
	}
}

func (s *SmoothWRR) UpdateServers(servers []*models.Server) {
	s.Lock()
	defer s.Unlock()
	s.servers = servers
	s.cw = make([]int, len(servers))
}
