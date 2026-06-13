package xor

import (
	"net"
	"sync"
)

func Length(length *uint32) uint32 {
	if length == nil {
		return 0
	}
	return *length
}

func Bytes(p []byte, key byte, length uint32) {
	if length > 0 && int(length) < len(p) {
		p = p[:length]
	}
	for index := range p {
		p[index] ^= key
	}
}

type StreamConn struct {
	net.Conn
	key            byte
	length         uint32
	readRemaining  int64
	writeRemaining int64
	readAccess     sync.Mutex
	writeAccess    sync.Mutex
}

func NewStreamConn(conn net.Conn, key byte, length uint32) net.Conn {
	return &StreamConn{
		Conn:           conn,
		key:            key,
		length:         length,
		readRemaining:  int64(length),
		writeRemaining: int64(length),
	}
}

func (c *StreamConn) Read(p []byte) (n int, err error) {
	n, err = c.Conn.Read(p)
	c.readAccess.Lock()
	bytesStream(p[:n], c.key, c.length, &c.readRemaining)
	c.readAccess.Unlock()
	return
}

func (c *StreamConn) Write(p []byte) (n int, err error) {
	buffer := make([]byte, len(p))
	copy(buffer, p)
	c.writeAccess.Lock()
	bytesStreamWithRemaining(buffer, c.key, c.length, c.writeRemaining)
	n, err = c.Conn.Write(buffer)
	advanceStreamRemaining(c.length, &c.writeRemaining, n)
	c.writeAccess.Unlock()
	return
}

type DatagramConn struct {
	net.Conn
	key    byte
	length uint32
}

func NewDatagramConn(conn net.Conn, key byte, length uint32) net.Conn {
	return &DatagramConn{
		Conn:   conn,
		key:    key,
		length: length,
	}
}

func (c *DatagramConn) Read(p []byte) (n int, err error) {
	n, err = c.Conn.Read(p)
	Bytes(p[:n], c.key, c.length)
	return
}

func (c *DatagramConn) Write(p []byte) (n int, err error) {
	buffer := make([]byte, len(p))
	copy(buffer, p)
	Bytes(buffer, c.key, c.length)
	return c.Conn.Write(buffer)
}

type PacketConn struct {
	net.PacketConn
	key    byte
	length uint32
}

func NewPacketConn(conn net.PacketConn, key byte, length uint32) net.PacketConn {
	return &PacketConn{
		PacketConn: conn,
		key:        key,
		length:     length,
	}
}

func (c *PacketConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	n, addr, err = c.PacketConn.ReadFrom(p)
	Bytes(p[:n], c.key, c.length)
	return
}

func (c *PacketConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	buffer := make([]byte, len(p))
	copy(buffer, p)
	Bytes(buffer, c.key, c.length)
	return c.PacketConn.WriteTo(buffer, addr)
}

func bytesStream(p []byte, key byte, length uint32, remaining *int64) {
	bytesStreamWithRemaining(p, key, length, *remaining)
	advanceStreamRemaining(length, remaining, len(p))
}

func bytesStreamWithRemaining(p []byte, key byte, length uint32, remaining int64) {
	if length == 0 {
		Bytes(p, key, 0)
		return
	}
	if remaining <= 0 {
		return
	}
	if int(remaining) < len(p) {
		p = p[:int(remaining)]
	}
	for index := range p {
		p[index] ^= key
	}
}

func advanceStreamRemaining(length uint32, remaining *int64, written int) {
	if length == 0 || written <= 0 || *remaining <= 0 {
		return
	}
	if int64(written) > *remaining {
		*remaining = 0
	} else {
		*remaining -= int64(written)
	}
}
