package strategy

// type LeastConnections struct {
// 	servers []*models.Server
// 	sync.RWMutex
// }

// func (s *LeastConnections) UpdateServers(servers []*models.Server) {
// 	s.Lock()
// 	defer s.Unlock()
// 	s.servers = servers
// }

// func (s *LeastConnections) Next(w http.ResponseWriter, r *http.Request) *models.Server {
// 	if len(s.servers) == 0 {
// 		return nil
// 	}

// 	s.Lock()
// 	defer s.Unlock()

// 	minIdx := -1
// 	minConn := int32(^uint32(0) >> 1)

// 	for i, srv := range s.servers {
// 		if srv.IsHealthy() {
// 			conns := srv.ConnCount.Load()
// 			if conns < minConn {
// 				minConn = conns
// 				minIdx = i
// 			}
// 		}
// 	}

// 	if minIdx == -1 {
// 		return nil
// 	}

// 	return s.servers[minIdx]
// }

// func NewLeastConnectionsStrategy(servers []*models.Server) ILBStrategy {
// 	return &LeastConnections{
// 		servers: servers,
// 	}
// }
