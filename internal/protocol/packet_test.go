package protocol

import (
	"testing"
	"time"
)

func TestPacketEncode(t *testing.T) {
	var nodeUUID [16]byte
	copy(nodeUUID[:], "test-node-uuid-01")

	pkt := &Packet{
		Version:    Version,
		NodeUUID:   nodeUUID,
		Timestamp:  1234567890123456789,
		StatusCode: 0,
	}

	data, err := pkt.Encode()
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if len(data) != PacketSize {
		t.Errorf("Encode() length = %d, want %d", len(data), PacketSize)
	}

	// Verify version
	if data[0] != Version {
		t.Errorf("Encode() version = %d, want %d", data[0], Version)
	}

	// Verify UUID
	for i := 0; i < 16; i++ {
		if data[1+i] != nodeUUID[i] {
			t.Errorf("Encode() UUID[%d] = %d, want %d", i, data[1+i], nodeUUID[i])
		}
	}

	// Verify status code
	if data[25] != 0 {
		t.Errorf("Encode() status code = %d, want 0", data[25])
	}

	// Verify checksum is present (last 4 bytes should not be all zeros)
	hasChecksum := false
	for i := PacketDataSize; i < PacketSize; i++ {
		if data[i] != 0 {
			hasChecksum = true
			break
		}
	}
	if !hasChecksum {
		t.Error("Encode() checksum is missing or zero")
	}
}

func TestPacketDecode(t *testing.T) {
	var nodeUUID [16]byte
	copy(nodeUUID[:], "test-node-uuid-02")

	pkt := &Packet{
		Version:    Version,
		NodeUUID:   nodeUUID,
		Timestamp:  9876543210987654,
		StatusCode: 1,
	}

	data, err := pkt.Encode()
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	decoded, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if decoded.Version != Version {
		t.Errorf("Decode() version = %d, want %d", decoded.Version, Version)
	}

	if decoded.NodeUUID != nodeUUID {
		t.Errorf("Decode() NodeUUID = %v, want %v", decoded.NodeUUID, nodeUUID)
	}

	if decoded.Timestamp != pkt.Timestamp {
		t.Errorf("Decode() Timestamp = %d, want %d", decoded.Timestamp, pkt.Timestamp)
	}

	if decoded.StatusCode != 1 {
		t.Errorf("Decode() StatusCode = %d, want 1", decoded.StatusCode)
	}
}

func TestPacketDecodeInvalidSize(t *testing.T) {
	invalidData := make([]byte, 20) // Wrong size

	_, err := Decode(invalidData)
	if err == nil {
		t.Error("Decode() should return error for invalid packet size")
	}
}

func TestPacketDecodeCorruptedChecksum(t *testing.T) {
	var nodeUUID [16]byte
	copy(nodeUUID[:], "test-node-uuid-03")

	pkt := &Packet{
		Version:    Version,
		NodeUUID:   nodeUUID,
		Timestamp:  time.Now().UnixNano(),
		StatusCode: 0,
	}

	data, err := pkt.Encode()
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	// Corrupt the checksum
	data[PacketSize-1] ^= 0xFF

	_, err = Decode(data)
	if err == nil {
		t.Error("Decode() should return error for corrupted checksum")
	}
}

func TestPacketRoundTrip(t *testing.T) {
	var nodeUUID [16]byte
	copy(nodeUUID[:], "round-trip-test")

	testCases := []struct {
		name       string
		version    uint8
		nodeUUID   [16]byte
		timestamp  int64
		statusCode uint8
	}{
		{"OK status", Version, nodeUUID, 1000000000, 0},
		{"Warn status", Version, nodeUUID, 2000000000, 1},
		{"Critical status", Version, nodeUUID, 3000000000, 2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pkt := &Packet{
				Version:    tc.version,
				NodeUUID:   tc.nodeUUID,
				Timestamp:  tc.timestamp,
				StatusCode: tc.statusCode,
			}

			data, err := pkt.Encode()
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}

			decoded, err := Decode(data)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}

			if decoded.Version != tc.version {
				t.Errorf("Version = %d, want %d", decoded.Version, tc.version)
			}
			if decoded.NodeUUID != tc.nodeUUID {
				t.Errorf("NodeUUID = %v, want %v", decoded.NodeUUID, tc.nodeUUID)
			}
			if decoded.Timestamp != tc.timestamp {
				t.Errorf("Timestamp = %d, want %d", decoded.Timestamp, tc.timestamp)
			}
			if decoded.StatusCode != tc.statusCode {
				t.Errorf("StatusCode = %d, want %d", decoded.StatusCode, tc.statusCode)
			}
		})
	}
}

func TestNewPacket(t *testing.T) {
	var nodeUUID [16]byte
	copy(nodeUUID[:], "new-packet-test")

	before := time.Now()
	pkt := NewPacket(nodeUUID, 1)
	after := time.Now()

	if pkt.Version != Version {
		t.Errorf("NewPacket() Version = %d, want %d", pkt.Version, Version)
	}

	if pkt.NodeUUID != nodeUUID {
		t.Errorf("NewPacket() NodeUUID = %v, want %v", pkt.NodeUUID, nodeUUID)
	}

	if pkt.StatusCode != 1 {
		t.Errorf("NewPacket() StatusCode = %d, want 1", pkt.StatusCode)
	}

	// Timestamp should be between before and after
	if pkt.Timestamp < before.UnixNano() || pkt.Timestamp > after.UnixNano() {
		t.Errorf("NewPacket() Timestamp = %d, should be between %d and %d",
			pkt.Timestamp, before.UnixNano(), after.UnixNano())
	}
}
