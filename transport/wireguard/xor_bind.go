package wireguard

import (
	"context"
	"net"
	"net/netip"
	"sync"

	boxXOR "github.com/sagernet/sing-box/common/xor"
	"github.com/sagernet/sing/common"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/wireguard-go/conn"
)

var _ conn.Bind = (*XORBind)(nil)

type XORBind struct {
	ctx                 context.Context
	listenAddress       netip.Addr
	listenPort          uint16
	key                 byte
	length              uint32
	reservedForEndpoint map[netip.AddrPort][3]uint8
	access              sync.Mutex
	conn                *net.UDPConn
}

func NewXORBind(ctx context.Context, listenAddress netip.Addr, listenPort uint16, key byte, length uint32) *XORBind {
	return &XORBind{
		ctx:                 ctx,
		listenAddress:       listenAddress,
		listenPort:          listenPort,
		key:                 key,
		length:              length,
		reservedForEndpoint: make(map[netip.AddrPort][3]uint8),
	}
}

func (b *XORBind) Open(port uint16) (fns []conn.ReceiveFunc, actualPort uint16, err error) {
	b.access.Lock()
	defer b.access.Unlock()
	if b.conn != nil {
		return nil, 0, conn.ErrBindAlreadyOpen
	}
	if b.listenPort != 0 {
		port = b.listenPort
	}
	listenAddress := b.listenAddress
	if !listenAddress.IsValid() {
		listenAddress = netip.IPv4Unspecified()
	}
	bindAddr := M.SocksaddrFrom(listenAddress, port)
	var listenConfig net.ListenConfig
	packetConn, err := listenConfig.ListenPacket(b.ctx, M.NetworkFromNetAddr(N.NetworkUDP, bindAddr.Addr), bindAddr.String())
	if err != nil {
		return nil, 0, err
	}
	udpConn := packetConn.(*net.UDPConn)
	b.conn = udpConn
	actualAddr := M.SocksaddrFromNet(udpConn.LocalAddr()).Unwrap()
	return []conn.ReceiveFunc{b.receive}, actualAddr.Port, nil
}

func (b *XORBind) receive(packets [][]byte, sizes []int, eps []conn.Endpoint) (count int, err error) {
	b.access.Lock()
	udpConn := b.conn
	b.access.Unlock()
	if udpConn == nil {
		return 0, net.ErrClosed
	}
	n, addr, err := udpConn.ReadFromUDPAddrPort(packets[0])
	if err != nil {
		return 0, err
	}
	packet := packets[0][:n]
	boxXOR.Bytes(packet, b.key, b.length)
	if n > 3 {
		clear(packet[1:4])
	}
	sizes[0] = n
	eps[0] = remoteEndpoint(addr)
	return 1, nil
}

func (b *XORBind) Close() error {
	b.access.Lock()
	defer b.access.Unlock()
	conn := b.conn
	b.conn = nil
	return common.Close(common.PtrOrNil(conn))
}

func (b *XORBind) SetMark(mark uint32) error {
	return nil
}

func (b *XORBind) Send(bufs [][]byte, ep conn.Endpoint, offset int) error {
	destination := netip.AddrPort(ep.(remoteEndpoint))
	b.access.Lock()
	udpConn := b.conn
	reserved, hasReserved := b.reservedForEndpoint[destination]
	b.access.Unlock()
	if udpConn == nil {
		return net.ErrClosed
	}
	for _, buf := range bufs {
		if offset > 0 {
			buf = buf[offset:]
		}
		packet := make([]byte, len(buf))
		copy(packet, buf)
		if hasReserved && len(packet) > 3 {
			copy(packet[1:4], reserved[:])
		}
		boxXOR.Bytes(packet, b.key, b.length)
		_, err := udpConn.WriteToUDPAddrPort(packet, destination)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *XORBind) ParseEndpoint(s string) (conn.Endpoint, error) {
	ap, err := netip.ParseAddrPort(s)
	if err != nil {
		return nil, err
	}
	return remoteEndpoint(ap), nil
}

func (b *XORBind) BatchSize() int {
	return 1
}

func (b *XORBind) SetReservedForEndpoint(destination netip.AddrPort, reserved [3]byte) {
	b.access.Lock()
	defer b.access.Unlock()
	b.reservedForEndpoint[destination] = reserved
}
