package protocol

import (
	"encoding/binary"
	"errors"
	"time"
)

const (
	PacketSize = 26
	Version    = 1
)

// Packet represents a 26-byte heartbeat packet
type Packet struct {
	Version   uint8
	NodeUUID  [16]byte
	Timestamp int64
	StatusCode uint8
}

// Encode encodes a packet into exactly 26 bytes
func (p *Packet) Encode() ([]byte, error) {
	buf := make([]byte, PacketSize)
	
	buf[0] = p.Version
	copy(buf[1:17], p.NodeUUID[:])
	binary.BigEndian.PutUint64(buf[17:25], uint64(p.Timestamp))
	buf[25] = p.StatusCode
	
	return buf, nil
}

// Decode decodes a 26-byte buffer into a packet
func Decode(data []byte) (*Packet, error) {
	if len(data) != PacketSize {
		return nil, errors.New("invalid packet size")
	}
	
	p := &Packet{
		Version:    data[0],
		Timestamp:  int64(binary.BigEndian.Uint64(data[17:25])),
		StatusCode: data[25],
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
