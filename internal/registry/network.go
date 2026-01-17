package registry

import (
	"log"
	"net"
	"runtime"
	"sync"

	"github.com/rafaelmarinho/pulsecheck/internal/protocol"
)

// packetJob represents a packet to be processed
type packetJob struct {
	data []byte
	addr *net.UDPAddr
}

// UDPNode represents a UDP network node
type UDPNode struct {
	conn         *net.UDPConn
	monitor      *Monitor
	nodeUUID     [16]byte
	peers        map[string]*net.UDPAddr
	peersMu      sync.RWMutex
	stopChan     chan struct{}
	packetChan   chan packetJob
	workerWg     sync.WaitGroup
	bufferPool   sync.Pool
	workerCount  int
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
	
	workerCount := runtime.NumCPU()
	if workerCount < 2 {
		workerCount = 2 // Minimum 2 workers
	}
	
	// Channel buffer size: allow some queuing during traffic spikes
	// Buffer size of 2x worker count provides headroom
	packetChanSize := workerCount * 2
	
	node := &UDPNode{
		conn:        conn,
		monitor:     monitor,
		nodeUUID:    nodeUUID,
		peers:       make(map[string]*net.UDPAddr),
		stopChan:    make(chan struct{}),
		packetChan:  make(chan packetJob, packetChanSize),
		workerCount: workerCount,
	}
	
	// Initialize buffer pool for receive buffers
	node.bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, protocol.PacketSize)
		},
	}
	
	return node, nil
}

// Start begins listening for UDP packets
func (u *UDPNode) Start() {
	log.Printf("UDP listener started on %s (workers: %d)", u.conn.LocalAddr(), u.workerCount)
	
	// Start worker pool
	u.startWorkers()
	
	// Main receive loop
	for {
		select {
		case <-u.stopChan:
			// Close packet channel to signal workers to stop
			close(u.packetChan)
			// Wait for all workers to finish
			u.workerWg.Wait()
			return
		default:
			// Get buffer from pool
			buf := u.bufferPool.Get().([]byte)
			
			n, addr, err := u.conn.ReadFromUDP(buf)
			if err != nil {
				// Return buffer to pool on error
				u.bufferPool.Put(buf)
				continue
			}
			
			if n != protocol.PacketSize {
				// Return buffer to pool if packet size is wrong
				u.bufferPool.Put(buf)
				continue
			}
			
			// Allocate packet data (26 bytes - minimal allocation)
			// We need a copy because buf will be returned to pool and reused
			packetData := make([]byte, protocol.PacketSize)
			copy(packetData, buf[:n])
			
			// Return receive buffer to pool immediately for reuse
			u.bufferPool.Put(buf)
			
			// Send to worker pool (non-blocking with buffered channel)
			select {
			case u.packetChan <- packetJob{data: packetData, addr: addr}:
				// Successfully queued
			default:
				// Channel full - drop packet to prevent blocking
				// In high-traffic scenarios, this prevents memory buildup
				log.Printf("Packet channel full, dropping packet from %s", addr)
			}
		}
	}
}

// startWorkers starts the worker pool goroutines
func (u *UDPNode) startWorkers() {
	for i := 0; i < u.workerCount; i++ {
		u.workerWg.Add(1)
		go u.worker(i)
	}
}

// worker processes packets from the channel
func (u *UDPNode) worker(id int) {
	defer u.workerWg.Done()
	
	for job := range u.packetChan {
		u.handlePacket(job.data, job.addr)
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
