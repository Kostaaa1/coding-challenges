package strategy

import (
	"sync"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type SWItem struct {
	server        *models.Server
	currentWeight int
}

type SW struct {
	servers []*SWItem
	sync.RWMutex
}

func (s *SW) Next() (peer *models.Server) {
	s.Lock()
	defer s.Unlock()

	if len(s.servers) == 0 {
		return nil
	}

	if len(s.servers) == 1 {
		return s.servers[0].server
	}

	var total int
	var found bool
	var selected *SWItem

	for _, item := range s.servers {
		if !item.server.IsHealthy() {
			continue
		}

		item.currentWeight += item.server.Weight
		total += item.server.Weight
		if !found || item.currentWeight > selected.currentWeight {
			selected = item
			found = true
		}
	}

	if selected != nil {
		selected.currentWeight -= total
		peer = selected.server
	}

	return peer
}

func (s *SW) UpdateServers(servers []*models.Server) {
	s.Lock()
	defer s.Unlock()
	s.servers = toSWItems(servers)
}

func toSWItems(servers []*models.Server) []*SWItem {
	swItems := make([]*SWItem, len(servers))
	for i, srv := range servers {
		swItems[i] = &SWItem{server: srv}
	}
	return swItems
}

func NewSmoothWRRStrategy(servers []*models.Server) ILBStrategy {
	return &SW{
		servers: toSWItems(servers),
	}
}
