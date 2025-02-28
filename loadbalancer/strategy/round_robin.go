package strategy

import (
	"net/http"
	"sync/atomic"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type RoundRobin struct {
	index   atomic.Int32
	servers []*models.Server
}

func NewRoundRobinStrategy(servers []*models.Server) ILBStrategy {
	return &RoundRobin{servers: servers}
}

func (s *RoundRobin) Next(w http.ResponseWriter, r *http.Request) *models.Server {
	if len(s.servers) == 0 {
		return nil
	}
	idx := int(s.index.Add(1)-1) % len(s.servers)
	return s.servers[idx]
}
