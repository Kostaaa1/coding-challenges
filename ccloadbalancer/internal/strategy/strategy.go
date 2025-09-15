package strategy

import (
	"errors"

	"github.com/Kostaaa1/loadbalancer/internal/config"
	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type ILBStrategy interface {
	Next() *models.Server // it always needs to return healthy server or nil
	UpdateServers(servers []*models.Server)
}

var (
	RoundRobinStrategy         = "round_robin"
	WeightedRoundRobinStrategy = "weighted_round_robin"
	SmoothWeightedRoundRobin   = "smooth_weighted_round_robin"
	StickySessionStrategy      = "sticky_session"
	RandomStrategy             = "random"
	// LeastConnectionsStrategy   = "leact_connections"
	// LeastConnectionsStrategy   = "ip_hash"
)

func GetFromConfig(cfg *config.Config) (ILBStrategy, error) {
	servers := cfg.Servers
	switch cfg.Strategy {
	case RoundRobinStrategy:
		return NewRoundRobinStrategy(servers), nil
	case WeightedRoundRobinStrategy:
		return NewWRRStrategy(servers), nil
	case SmoothWeightedRoundRobin:
		return NewSmoothWRRStrategy(servers), nil
	case RandomStrategy:
		return NewRandomStrategy(servers), nil
	// case StickySessionStrategy:
	// 	return NewStickySessionStrategy(), nil
	// case LeastConnectionsStrategy:
	// 	return NewLeastConnectionsStrategy(servers), nil
	default:
		return nil, errors.New("provided strategy is not valid")
	}
}
