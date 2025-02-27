package strategy

import (
	"net/http"
	"sync"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type LeastConnections struct {
	servers []*models.Server
	sync.RWMutex
}

func (s *LeastConnections) Next(w http.ResponseWriter, r *http.Request) *models.Server {
	if len(s.servers) == 0 {
		return nil
	}

	minIdx := -1
	minConn := int32(^uint32(0) >> 1)

	for i, srv := range s.servers {
		if srv.Healthy {
			conns := srv.ConnCount.Load()
			if conns < minConn {
				minConn = conns
				minIdx = i
			}
		}
	}

	if minIdx == -1 {
		return nil
	}

	return s.servers[minIdx]
}

func NewLeastConnectionsStrategy(servers []*models.Server) ILBStrategy {
	return &LeastConnections{
		servers: servers,
	}
}
