package protocol

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"time"
)

const (
	PacketSize     = 30  // 26 bytes data + 4 bytes CRC32 checksum
	PacketDataSize = 26  // Size of data before checksum
	Version        = 1
)

// Packet represents a 30-byte heartbeat packet (26 bytes data + 4 bytes CRC32)
type Packet struct {
	Version    uint8
	NodeUUID   [16]byte
	Timestamp  int64
	StatusCode uint8
	Checksum   uint32 // CRC32 checksum of the first 26 bytes
}

// Encode encodes a packet into exactly 30 bytes (26 bytes data + 4 bytes CRC32)
func (p *Packet) Encode() ([]byte, error) {
	buf := make([]byte, PacketSize)
	
	// Pack data fields (first 26 bytes)
	buf[0] = p.Version
	copy(buf[1:17], p.NodeUUID[:])
	binary.BigEndian.PutUint64(buf[17:25], uint64(p.Timestamp))
	buf[25] = p.StatusCode
	
	// Calculate CRC32 checksum over the data portion (first 26 bytes)
	checksum := crc32.ChecksumIEEE(buf[0:PacketDataSize])
	p.Checksum = checksum
	
	// Append checksum (last 4 bytes)
	binary.BigEndian.PutUint32(buf[PacketDataSize:PacketSize], checksum)
	
	return buf, nil
}

// Decode decodes a 30-byte buffer into a packet and verifies CRC32 checksum
func Decode(data []byte) (*Packet, error) {
	if len(data) != PacketSize {
		return nil, errors.New("invalid packet size")
	}
	
	// Extract checksum from last 4 bytes
	receivedChecksum := binary.BigEndian.Uint32(data[PacketDataSize:PacketSize])
	
	// Calculate expected checksum over data portion (first 26 bytes)
	expectedChecksum := crc32.ChecksumIEEE(data[0:PacketDataSize])
	
	// Verify checksum
	if receivedChecksum != expectedChecksum {
		return nil, errors.New("packet checksum verification failed - packet may be corrupted")
	}
	
	// Decode packet fields
	p := &Packet{
		Version:    data[0],
		Timestamp:  int64(binary.BigEndian.Uint64(data[17:25])),
		StatusCode: data[25],
		Checksum:   receivedChecksum,
	}
	
	copy(p.NodeUUID[:], data[1:17])
	
	return p, nil
}

// NewPacket creates a new packet with current timestamp
func NewPacket(nodeUUID [16]byte, statusCode uint8) *Packet {
	return &Packet{
		Version:    Version,
		NodeUUID:   nodeUUID,
		Timestamp:  time.Now().UnixNano(),
		StatusCode: statusCode,
	}
}
