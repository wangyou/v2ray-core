package point

import (
	"github.com/v2ray/v2ray-core/app"
	"github.com/v2ray/v2ray-core/common/log"
	v2net "github.com/v2ray/v2ray-core/common/net"
	"github.com/v2ray/v2ray-core/common/retry"
	"github.com/v2ray/v2ray-core/proxy/common/connhandler"
)

type InboundConnectionHandlerWithPort struct {
	port    v2net.Port
	handler connhandler.InboundConnectionHandler
}

// Handler for inbound detour connections.
type InboundDetourHandler struct {
	space  app.Space
	config InboundDetourConfig
	ich    []*InboundConnectionHandlerWithPort
}

func (this *InboundDetourHandler) Initialize() error {
	ichFactory := connhandler.GetInboundConnectionHandlerFactory(this.config.Protocol())
	if ichFactory == nil {
		log.Error("Unknown inbound connection handler factory %s", this.config.Protocol())
		return BadConfiguration
	}

	ports := this.config.PortRange()
	this.ich = make([]*InboundConnectionHandlerWithPort, 0, ports.To()-ports.From()+1)
	for i := ports.From(); i <= ports.To(); i++ {
		ichConfig := this.config.Settings()
		ich, err := ichFactory.Create(this.space, ichConfig)
		if err != nil {
			log.Error("Failed to create inbound connection handler: %v", err)
			return err
		}
		this.ich = append(this.ich, &InboundConnectionHandlerWithPort{
			port:    i,
			handler: ich,
		})
	}
	return nil
}

// Starts the inbound connection handler.
func (this *InboundDetourHandler) Start() error {
	for _, ich := range this.ich {
		err := retry.Timed(100 /* times */, 100 /* ms */).On(func() error {
			err := ich.handler.Listen(ich.port)
			if err != nil {
				log.Error("Failed to start inbound detour on port %d: %v", ich.port, err)
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
