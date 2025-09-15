package strategy

import (
	"math/rand"
	"sync"
	"time"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type Random struct {
	servers []*models.Server
	total   int
	sync.RWMutex
}

func (s *Random) Next() *models.Server {
	s.RLock()
	defer s.RUnlock()

	if len(s.servers) == 0 {
		return nil
	}

	random := rand.New(rand.NewSource(time.Now().UnixNano()))

	if s.total > 0 {
		rInt := random.Intn(s.total)
		for _, srv := range s.servers {
			if srv.IsHealthy() {
				if rInt < srv.Weight {
					return srv
				}
				rInt -= srv.Weight
			}
		}
	} else {
		var healthy []*models.Server
		for _, srv := range s.servers {
			if srv.IsHealthy() {
				healthy = append(healthy, srv)
			}
		}

		if len(healthy) == 0 {
			return nil
		}

		rInt := random.Intn(len(healthy))
		return healthy[rInt]
	}

	return nil
}

func (s *Random) UpdateServers(servers []*models.Server) {
	s.Lock()
	s.servers = servers
	s.total = calcTotal(servers)
	s.Unlock()
}

func calcTotal(servers []*models.Server) int {
	total := 0
	for _, srv := range servers {
		if srv.IsHealthy() {
			total += srv.Weight
		}
	}
	return total
}

func NewRandomStrategy(servers []*models.Server) ILBStrategy {
	return &Random{
		servers: servers,
		total:   calcTotal(servers),
	}
}
