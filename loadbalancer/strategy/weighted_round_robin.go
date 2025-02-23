package strategy

import (
	"net/http"
	"sort"
	"sync"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type WeightedRoundRobinStrategy struct {
	servers     []*models.Server
	weightsMap  map[string]int
	index       int
	totalWeight int
	sync.Mutex
}

func (s *WeightedRoundRobinStrategy) Next(w http.ResponseWriter, r *http.Request) *models.Server {
	s.Lock()
	defer s.Unlock()

	if len(s.servers) == 0 {
		return nil
	}

	for {
		srv := s.servers[s.index]
		if s.weightsMap[srv.Name] > 0 {
			s.weightsMap[srv.Name]--
			s.index = (s.index + 1) % len(s.servers)
			return srv
		}

		if s.allWeightsZero() {
			s.resetWeights()
		}

		s.index = (s.index + 1) % len(s.servers)
	}
}

func (s *WeightedRoundRobinStrategy) allWeightsZero() bool {
	for _, w := range s.weightsMap {
		if w > 0 {
			return false
		}
	}
	return true
}

func (s *WeightedRoundRobinStrategy) resetWeights() {
	for _, srv := range s.servers {
		s.weightsMap[srv.Name] = srv.Weight
	}
}

func NewWeightedRoundRobin(servers []*models.Server) ILBStrategy {
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Weight > servers[j].Weight
	})
	return &WeightedRoundRobinStrategy{
		servers:     servers,
		weightsMap:  newWeightsMap(servers),
		index:       0,
		totalWeight: sumWeights(servers),
	}
}

// UpdateServers updates the list of servers and resets weight mappings.
// func (s *WeightedRoundRobinStrategy) UpdateServers(servers []*models.Server) {
// 	s.Lock()
// 	defer s.Unlock()
// 	sort.Slice(servers, func(i, j int) bool {
// 		return servers[i].Weight > servers[j].Weight
// 	})
// 	s.servers = servers
// 	s.weightsMap = newWeightsMap(servers)
// 	s.index = 0
// 	s.totalWeight = sumWeights(servers)
// }

func newWeightsMap(servers []*models.Server) map[string]int {
	wm := make(map[string]int, len(servers))
	for _, srv := range servers {
		wm[srv.Name] = srv.Weight
	}
	return wm
}

func sumWeights(servers []*models.Server) int {
	total := 0
	for _, srv := range servers {
		total += srv.Weight
	}
	return total
}
