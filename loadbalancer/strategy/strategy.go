package strategy

import (
	"errors"
	"net/http"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type ILBStrategy interface {
	Next(w http.ResponseWriter, r *http.Request) *models.Server // it always needs to return healthy server or nil
}

var (
	RoundRobinStrategy         = "round_robin"
	WeightedRoundRobinStrategy = "weighted_round_robin"
	SmoothWeightedRoundRobin   = "round_robin"
	LeastConnectionsStrategy   = "leact_connections"
	StickySessionStrategy      = "sticky_session"
)

func GetLBStrategy(strategy string, servers []*models.Server) (ILBStrategy, error) {
	switch strategy {
	case RoundRobinStrategy:
		return NewRoundRobinStrategy(servers), nil
	case StickySessionStrategy:
		return NewStickySessionStrategy(), nil
	case WeightedRoundRobinStrategy:
		return NewWRRStrategy(servers), nil
	case SmoothWeightedRoundRobin:
		return NewSmoothWRRStrategy(servers), nil
	case LeastConnectionsStrategy:
		return NewLeastConnectionsStrategy(servers), nil
	default:
		return nil, errors.New("provided strategy is not valid")
	}
}
