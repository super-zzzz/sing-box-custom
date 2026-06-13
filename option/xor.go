package option

import "github.com/sagernet/sing/common/json/badoption"

type XOROptions struct {
	XORTo     *uint8  `json:"xor-to"`
	XORLength *uint32 `json:"xor-length,omitempty"`
}

type XOROutboundOptions struct {
	DialerOptions
	XOROptions
}

type WireGuardBindOptions struct {
	Type       string          `json:"type"`
	Listen     *badoption.Addr `json:"listen,omitempty"`
	ListenPort uint16          `json:"listen_port,omitempty"`
	XOROptions
}
