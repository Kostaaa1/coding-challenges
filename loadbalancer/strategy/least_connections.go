package strategy

import (
	"net/http"
	"sync"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type LeastConnections struct {
	servers []*models.Server
	conns   []int
	sync.RWMutex
}

func (s *LeastConnections) Next(w http.ResponseWriter, r *http.Request) *models.Server {
	s.Lock()
	defer s.Unlock()

	if len(s.servers) == 0 {
		return nil
	}

	minId := 0
	for i := 1; i < len(s.conns); i++ {
		if s.conns[i] < s.conns[minId] {
			minId = i
		}
	}

	return s.servers[minId]
}

func (s *LeastConnections) increment(id int) {
	s.Lock()
	s.conns[id]++
	s.Unlock()
}

func (s *LeastConnections) decrement(id int) {
	s.Lock()
	s.conns[id]--
	s.Unlock()
}

func NewLeastConnectionsStrategy(servers []*models.Server) ILBStrategy {
	return &LeastConnections{
		servers: servers,
		conns:   make([]int, len(servers)),
	}
}
