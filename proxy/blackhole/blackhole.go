package blackhole

import (
	"io/ioutil"

	"github.com/v2ray/v2ray-core/app"
	v2io "github.com/v2ray/v2ray-core/common/io"
	v2net "github.com/v2ray/v2ray-core/common/net"
	"github.com/v2ray/v2ray-core/proxy"
	"github.com/v2ray/v2ray-core/proxy/internal"
	"github.com/v2ray/v2ray-core/transport/ray"
)

// BlackHole is an outbound connection that sliently swallow the entire payload.
type BlackHole struct {
}

func NewBlackHole() *BlackHole {
	return &BlackHole{}
}

func (this *BlackHole) Dispatch(firstPacket v2net.Packet, ray ray.OutboundRay) error {
	if chunk := firstPacket.Chunk(); chunk != nil {
		chunk.Release()
	}

	close(ray.OutboundOutput())
	if firstPacket.MoreChunks() {
		v2io.ChanToRawWriter(ioutil.Discard, ray.OutboundInput())
	}
	return nil
}

func init() {
	internal.MustRegisterOutboundHandlerCreator("blackhole",
		func(space app.Space, config interface{}) (proxy.OutboundHandler, error) {
			return NewBlackHole(), nil
		})
}
