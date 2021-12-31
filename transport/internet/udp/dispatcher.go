package udp

import (
	"context"

	"github.com/v2fly/v2ray-core/v4/common"
	"github.com/v2fly/v2ray-core/v4/common/buf"
	"github.com/v2fly/v2ray-core/v4/common/net"
)

type DispatcherI interface {
	common.Closable
	Dispatch(ctx context.Context, destination net.Destination, payload *buf.Buffer)
}
