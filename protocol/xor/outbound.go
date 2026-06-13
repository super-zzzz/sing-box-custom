package xor

import (
	"context"
	"net"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/common/dialer"
	boxXOR "github.com/sagernet/sing-box/common/xor"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/logger"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

func RegisterOutbound(registry *outbound.Registry) {
	outbound.Register[option.XOROutboundOptions](registry, C.TypeXOR, NewOutbound)
}

type Outbound struct {
	outbound.Adapter
	logger logger.ContextLogger
	dialer N.Dialer
	key    byte
	length uint32
}

func NewOutbound(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.XOROutboundOptions) (adapter.Outbound, error) {
	if options.XORTo == nil {
		return nil, E.New("missing xor-to")
	}
	outboundDialer, err := dialer.New(ctx, options.DialerOptions, true)
	if err != nil {
		return nil, err
	}
	return &Outbound{
		Adapter: outbound.NewAdapterWithDialerOptions(C.TypeXOR, tag, []string{N.NetworkTCP, N.NetworkUDP}, options.DialerOptions),
		logger:  logger,
		dialer:  outboundDialer,
		key:     *options.XORTo,
		length:  boxXOR.Length(options.XORLength),
	}, nil
}

func (h *Outbound) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	ctx, metadata := adapter.ExtendContext(ctx)
	metadata.Outbound = h.Tag()
	metadata.Destination = destination
	conn, err := h.dialer.DialContext(ctx, network, destination)
	if err != nil {
		return nil, err
	}
	switch N.NetworkName(network) {
	case N.NetworkTCP:
		h.logger.InfoContext(ctx, "outbound XOR connection to ", destination)
		return boxXOR.NewStreamConn(conn, h.key, h.length), nil
	case N.NetworkUDP:
		h.logger.InfoContext(ctx, "outbound XOR packet connection to ", destination)
		return boxXOR.NewDatagramConn(conn, h.key, h.length), nil
	default:
		conn.Close()
		return nil, E.Extend(N.ErrUnknownNetwork, network)
	}
}

func (h *Outbound) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	ctx, metadata := adapter.ExtendContext(ctx)
	metadata.Outbound = h.Tag()
	metadata.Destination = destination
	h.logger.InfoContext(ctx, "outbound XOR packet connection")
	conn, err := h.dialer.ListenPacket(ctx, destination)
	if err != nil {
		return nil, err
	}
	return boxXOR.NewPacketConn(conn, h.key, h.length), nil
}
