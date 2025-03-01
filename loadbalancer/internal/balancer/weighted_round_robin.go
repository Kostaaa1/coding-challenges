package balancer

import (
	"net/http"
	"sort"
	"sync"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type WRR struct {
	servers []*models.Server
	cw      []int
	index   int
	sync.RWMutex
}

func (s *WRR) Next(w http.ResponseWriter, r *http.Request) *models.Server {
	s.Lock()
	defer s.Unlock()

	if len(s.servers) == 0 {
		return nil
	}

	checked := 0

	for checked < len(s.servers) {
		srv := s.servers[s.index]
		if srv.Healthy && s.cw[s.index] > 0 {
			s.cw[s.index]--
			return srv
		}

		checked++
		s.index = (s.index + 1) % len(s.servers)

		if s.isCycleOver() {
			s.resetWeights()
		}
	}

	// fallback
	for _, srv := range s.servers {
		if srv.Healthy {
			return srv
		}
	}

	return nil
}

func (s *WRR) resetWeights() {
	for i, srv := range s.servers {
		s.cw[i] = srv.Weight
	}
}

func (s *WRR) isCycleOver() bool {
	for _, weight := range s.cw {
		if weight > 0 {
			return false
		}
	}
	return true
}

func NewWRRStrategy(servers []*models.Server) ILBStrategy {
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Weight > servers[j].Weight
	})
	cw := make([]int, len(servers))
	for i, srv := range servers {
		cw[i] = srv.Weight
	}
	return &WRR{
		servers: servers,
		cw:      cw,
	}
}

// package balancer
// import (
// 	"net/http"
// 	"sort"
// 	"sync"
// 	"sync/atomic"
// 	"github.com/Kostaaa1/loadbalancer/internal/models"
// )
// type WRR struct {
// 	servers      []*models.Server
// 	cw           []atomic.Int32
// 	index        atomic.Int32
// 	resetLock    sync.Mutex
// 	resetCounter atomic.Int32 // Tracks when resets happen
// }
// func (s *WRR) Next(w http.ResponseWriter, r *http.Request) *models.Server {
// 	if len(s.servers) == 0 {
// 		return nil
// 	}
// 	// Remember current reset counter to detect if a reset occurs
// 	initialResetCount := s.resetCounter.Load()
// 	// Try to find a server using lock-free approach first
// 	for attempt := 0; attempt < 2; attempt++ {
// 		// Get the next server index (atomic)
// 		currentIndex := int(s.index.Add(1)) % len(s.servers)
// 		srv := s.servers[currentIndex]
// 		// Try to decrement weight if positive
// 		for {
// 			currentWeight := s.cw[currentIndex].Load()
// 			if currentWeight <= 0 {
// 				// No capacity left for this server
// 				break
// 			}
// 			// Try to claim this server by atomically decrementing its weight
// 			if s.cw[currentIndex].CompareAndSwap(currentWeight, currentWeight-1) {
// 				if srv.Healthy {
// 					return srv
// 				}
// 				break
// 			}
// 			// Weight was changed by another goroutine, retry
// 		}
// 		// If reset happened during our attempts, try again
// 		if s.resetCounter.Load() != initialResetCount {
// 			continue
// 		}
// 		// Check if cycle is over - but do this rarely to avoid contention
// 		// Only check when we've tried multiple servers without success
// 		if attempt > 0 {
// 			allZero := true
// 			for i := range s.cw {
// 				if s.cw[i].Load() > 0 {
// 					allZero = false
// 					break
// 				}
// 			}
// 			if allZero {
// 				// Try to acquire reset lock - only one goroutine will get this
// 				if s.resetLock.TryLock() {
// 					// Double-check that cycle is still over (might have changed)
// 					allZero = true
// 					for i := range s.cw {
// 						if s.cw[i].Load() > 0 {
// 							allZero = false
// 							break
// 						}
// 					}
// 					if allZero {
// 						// Reset all weights
// 						for i, srv := range s.servers {
// 							s.cw[i].Store(int32(srv.Weight))
// 						}
// 						// Increment reset counter to signal a reset happened
// 						s.resetCounter.Add(1)
// 					}
// 					s.resetLock.Unlock()
// 					// Try again with fresh weights
// 					continue
// 				}
// 			}
// 		}
// 	}
// 	// Fallback - linear search for any healthy server
// 	for _, srv := range s.servers {
// 		if srv.Healthy {
// 			return srv
// 		}
// 	}
// 	return nil
// }
// func NewWRRStrategy(servers []*models.Server) ILBStrategy {
// 	sort.Slice(servers, func(i, j int) bool {
// 		return servers[i].Weight > servers[j].Weight
// 	})
// 	cw := make([]atomic.Int32, len(servers))
// 	for i, srv := range servers {
// 		cw[i].Store(int32(srv.Weight))
// 	}
// 	return &WRR{
// 		servers: servers,
// 		cw:      cw,
// 	}
// }
