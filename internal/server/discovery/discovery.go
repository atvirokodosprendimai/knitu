package discovery

import (
	"log"
	"time"

	"github.com/atvirokodosprendimai/knitu/internal/db"
	"github.com/atvirokodosprendimai/knitu/internal/wgmesh"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Service handles the discovery of nodes from the wg-mesh network.
type Service struct {
	db       *gorm.DB
	wgClient *wgmesh.Client
	ticker   *time.Ticker
	stopCh   chan bool
}

// NewService creates a new discovery service.
func NewService(db *gorm.DB, wgClient *wgmesh.Client, interval time.Duration) *Service {
	return &Service{
		db:       db,
		wgClient: wgClient,
		ticker:   time.NewTicker(interval),
		stopCh:   make(chan bool),
	}
}

// Start begins the periodic syncing of nodes from wg-mesh.
func (s *Service) Start() {
	log.Println("[INFO] Starting wg-mesh discovery service...")
	go func() {
		// Sync immediately on start
		s.syncPeers()

		for {
			select {
			case <-s.ticker.C:
				s.syncPeers()
			case <-s.stopCh:
				log.Println("[INFO] Stopping discovery service.")
				s.ticker.Stop()
				return
			}
		}
	}()
}

// Stop halts the discovery service.
func (s *Service) Stop() {
	s.stopCh <- true
}

func (s *Service) syncPeers() {
	log.Println("[INFO] Syncing peers from wg-mesh...")
	peers, err := s.wgClient.GetPeers()
	if err != nil {
		log.Printf("[ERROR] Failed to get peers from wg-mesh: %v", err)
		return
	}

	if len(peers) == 0 {
		log.Println("[INFO] No peers returned from wg-mesh.")
		return
	}

	log.Printf("[INFO] Discovered %d peers in the mesh:", len(peers))
	for _, peer := range peers {
		log.Printf("  - Peer PubKey: %s, IP: %s", peer.PubKey, peer.MeshIP)
	}

	for _, peer := range peers {
		node := db.Node{
			NodeID: peer.PubKey,
			// Hostname is intentionally left empty. It will be populated
			// by the agent's first heartbeat, which is the source of truth for hostname.
		}

		// Upsert the node based on the public key (NodeID).
		// This just ensures the node record exists.
		result := s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "node_id"}},
			DoNothing: true,
		}).Create(&node)

		if result.Error != nil {
			// This error can still happen if the user hasn't deleted the old knit.db file
			// with the unique constraint on the empty hostname.
			log.Printf("[ERROR] Failed to create node record for %s: %v", peer.PubKey, result.Error)
		} else if result.RowsAffected > 0 {
			log.Printf("[INFO] Discovered and added new node from mesh: %s", peer.PubKey)
		}
	}
}
