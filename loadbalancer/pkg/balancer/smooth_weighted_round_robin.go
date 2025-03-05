package balancer

import (
	"net/http"
	"sync"
)

type BalancerItem interface {
	IsHealthy() bool
	GetWeight() int
}

type SWItem struct {
	item          BalancerItem
	currentWeight int
}

type SW struct {
	items []*SWItem
	sync.RWMutex
}

func (s *SW) Next(w http.ResponseWriter, r *http.Request) BalancerItem {
	s.Lock()
	defer s.Unlock()

	if len(s.items) == 0 {
		return nil
	}

	if len(s.items) == 1 {
		return s.items[0].item
	}

	var total int
	var found bool

	var maxPeer *SWItem

	for _, srv := range s.items {
		if !srv.item.IsHealthy() {
			continue
		}

		srv.currentWeight += srv.item.GetWeight()
		total += srv.item.GetWeight()
		if !found || srv.currentWeight > maxPeer.currentWeight {
			maxPeer = srv
			found = true
		}
	}

	if maxPeer != nil {
		maxPeer.currentWeight -= total
	}

	return maxPeer.item
}

func (s *SW) UpdateServers(servers []BalancerItem) {
	s.Lock()
	defer s.Unlock()
	s.items = initSWItems(servers)
}

func initSWItems(servers []BalancerItem) []*SWItem {
	sw := make([]*SWItem, len(servers))
	for i, srv := range servers {
		sw[i] = &SWItem{
			item:          srv,
			currentWeight: 0,
		}
	}
	return sw
}

type ILBStrategy interface {
	Next(w http.ResponseWriter, r *http.Request) BalancerItem
	UpdateServers(servers []BalancerItem)
}

func NewSmoothWRRStrategy(servers []BalancerItem) ILBStrategy {
	return &SW{
		items: initSWItems(servers),
	}
}
