package registry

import (
	"log"
	"net"
	"sync"

	"github.com/rafaelmarinho/pulsecheck/internal/protocol"
)

// UDPNode represents a UDP network node
type UDPNode struct {
	conn       *net.UDPConn
	monitor    *Monitor
	nodeUUID   [16]byte
	peers      map[string]*net.UDPAddr
	peersMu    sync.RWMutex
	stopChan   chan struct{}
}

// NewUDPNode creates a new UDP node
func NewUDPNode(port int, nodeUUID [16]byte, monitor *Monitor) (*UDPNode, error) {
	addr := &net.UDPAddr{
		Port: port,
		IP:   net.ParseIP("0.0.0.0"),
	}
	
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	
	return &UDPNode{
		conn:     conn,
		monitor:  monitor,
		nodeUUID: nodeUUID,
		peers:    make(map[string]*net.UDPAddr),
		stopChan: make(chan struct{}),
	}, nil
}

// Start begins listening for UDP packets
func (u *UDPNode) Start() {
	log.Printf("UDP listener started on %s", u.conn.LocalAddr())
	
	buf := make([]byte, protocol.PacketSize)
	
	for {
		select {
		case <-u.stopChan:
			return
		default:
			n, addr, err := u.conn.ReadFromUDP(buf)
			if err != nil {
				// Non-blocking: continue on error
				continue
			}
			
			if n != protocol.PacketSize {
				continue
			}
			
			// Handle packet in goroutine for non-blocking behavior
			go u.handlePacket(buf[:n], addr)
		}
	}
}

// handlePacket processes an incoming heartbeat packet
func (u *UDPNode) handlePacket(data []byte, addr *net.UDPAddr) {
	pkt, err := protocol.Decode(data)
	if err != nil {
		log.Printf("Failed to decode packet from %s: %v", addr, err)
		return
	}
	
	// Add peer to known peers
	addrStr := addr.String()
	u.peersMu.Lock()
	u.peers[addrStr] = addr
	u.peersMu.Unlock()
	
	// Update monitor with node info
	// Note: We don't have telemetry in the packet, so we use defaults
	// The status code tells us the health state
	u.monitor.UpdateWithStatus(addrStr, pkt.StatusCode, pkt.Timestamp)
}

// BroadcastHeartbeat sends a heartbeat packet to all known peers
func (u *UDPNode) BroadcastHeartbeat(statusCode uint8) error {
	pkt := protocol.NewPacket(u.nodeUUID, statusCode)
	data, err := pkt.Encode()
	if err != nil {
		return err
	}
	
	u.peersMu.RLock()
	peers := make([]*net.UDPAddr, 0, len(u.peers))
	for _, addr := range u.peers {
		peers = append(peers, addr)
	}
	u.peersMu.RUnlock()
	
	// If no peers, broadcast to local network
	if len(peers) == 0 {
		// Broadcast to subnet (optional, for discovery)
		return nil
	}
	
	// Send to all known peers
	for _, addr := range peers {
		_, err := u.conn.WriteToUDP(data, addr)
		if err != nil {
			log.Printf("Failed to send heartbeat to %s: %v", addr, err)
		}
	}
	
	return nil
}

// AddPeer adds a peer address to the known peers list
func (u *UDPNode) AddPeer(addrStr string) error {
	addr, err := net.ResolveUDPAddr("udp", addrStr)
	if err != nil {
		return err
	}
	
	u.peersMu.Lock()
	u.peers[addrStr] = addr
	u.peersMu.Unlock()
	
	return nil
}

// Conn returns the UDP connection (for getting local address)
func (u *UDPNode) Conn() *net.UDPConn {
	return u.conn
}

// Stop stops the UDP listener
func (u *UDPNode) Stop() {
	close(u.stopChan)
	u.conn.Close()
}
