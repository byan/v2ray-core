package packetaddr

import (
	"context"
	gonet "net"
	"sync"
	"time"

	"github.com/v2fly/v2ray-core/v4/common"
	"github.com/v2fly/v2ray-core/v4/common/buf"
	"github.com/v2fly/v2ray-core/v4/common/errors"
	"github.com/v2fly/v2ray-core/v4/common/net"
	"github.com/v2fly/v2ray-core/v4/features/routing"
	"github.com/v2fly/v2ray-core/v4/transport"
)

var errNotPacketConn = errors.New("not a packet connection")
var errUnsupported = errors.New("unsupported action")

func ToPacketAddrConn(link *transport.Link, dest net.Destination) (net.PacketConn, error) {
	if !dest.Address.Family().IsDomain() {
		return nil, errNotPacketConn
	}
	switch dest.Address.Domain() {
	case seqPacketMagicAddress:
		return &packetConnectionAdaptor{
			readerAccess: &sync.Mutex{},
			readerBuffer: nil,
			link:         link,
		}, nil
	default:
		return nil, errNotPacketConn
	}
}

func CreatePacketAddrConn(ctx context.Context, dispatcher routing.Dispatcher, isStream bool) (net.PacketConn, error) {
	if isStream {
		return nil, errUnsupported
	}
	packetDest := net.Destination{
		Address: net.DomainAddress(seqPacketMagicAddress),
		Port:    0,
		Network: net.Network_UDP,
	}
	link, err := dispatcher.Dispatch(ctx, packetDest)
	if err != nil {
		return nil, err
	}
	return &packetConnectionAdaptor{
		readerAccess: &sync.Mutex{},
		readerBuffer: nil,
		link:         link,
	}, nil
}

type packetConnectionAdaptor struct {
	readerAccess *sync.Mutex
	readerBuffer buf.MultiBuffer
	link         *transport.Link
}

func (c *packetConnectionAdaptor) ReadFrom(p []byte) (n int, addr gonet.Addr, err error) {
	c.readerAccess.Lock()
	defer c.readerAccess.Unlock()
	if c.readerBuffer.IsEmpty() {
		c.readerBuffer, err = c.link.Reader.ReadMultiBuffer()
	}
	c.readerBuffer, n = buf.SplitFirstBytes(c.readerBuffer, p)
	p, addr = ExtractAddressFromPacket(p)
	return
}

func (c *packetConnectionAdaptor) WriteTo(p []byte, addr gonet.Addr) (n int, err error) {
	payloadLen := len(p)
	p = AttachAddressToPacket(p, addr)
	buffer := buf.New()
	mb := buf.MultiBuffer{buffer}
	err = c.link.Writer.WriteMultiBuffer(mb)
	buf.ReleaseMulti(mb)
	if err != nil {
		return 0, err
	}
	return payloadLen, nil
}

func (c *packetConnectionAdaptor) Close() error {
	c.readerAccess.Lock()
	defer c.readerAccess.Unlock()
	c.readerBuffer = buf.ReleaseMulti(c.readerBuffer)
	return common.Interrupt(c.link)
}

func (c packetConnectionAdaptor) LocalAddr() gonet.Addr {
	return &gonet.UnixAddr{Name: "unsupported"}
}

func (c packetConnectionAdaptor) SetDeadline(t time.Time) error {
	return nil
}

func (c packetConnectionAdaptor) SetReadDeadline(t time.Time) error {
	return nil
}

func (c packetConnectionAdaptor) SetWriteDeadline(t time.Time) error {
	return nil
}
