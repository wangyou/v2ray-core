// Package point is a shell of V2Ray to run on various of systems.
// Point server is a full functionality proxying system. It consists of an inbound and an outbound
// connection, as well as any number of inbound and outbound detours. It provides a way internally
// to route network packets.
package point

import (
	"github.com/v2ray/v2ray-core/app"
	"github.com/v2ray/v2ray-core/app/dispatcher"
	"github.com/v2ray/v2ray-core/app/proxyman"
	"github.com/v2ray/v2ray-core/app/router"
	"github.com/v2ray/v2ray-core/common/log"
	v2net "github.com/v2ray/v2ray-core/common/net"
	"github.com/v2ray/v2ray-core/common/retry"
	"github.com/v2ray/v2ray-core/proxy"
	proxyrepo "github.com/v2ray/v2ray-core/proxy/repo"
	"github.com/v2ray/v2ray-core/transport/ray"
)

// Point shell of V2Ray.
type Point struct {
	port      v2net.Port
	ich       proxy.InboundHandler
	och       proxy.OutboundHandler
	idh       []InboundDetourHandler
	taggedIdh map[string]InboundDetourHandler
	odh       map[string]proxy.OutboundHandler
	router    router.Router
	space     *app.SpaceController
}

// NewPoint returns a new Point server based on given configuration.
// The server is not started at this point.
func NewPoint(pConfig *Config) (*Point, error) {
	var vpoint = new(Point)
	vpoint.port = pConfig.Port

	if pConfig.LogConfig != nil {
		logConfig := pConfig.LogConfig
		if len(logConfig.AccessLog) > 0 {
			err := log.InitAccessLogger(logConfig.AccessLog)
			if err != nil {
				return nil, err
			}
		}

		if len(logConfig.ErrorLog) > 0 {
			err := log.InitErrorLogger(logConfig.ErrorLog)
			if err != nil {
				return nil, err
			}
		}

		log.SetLogLevel(logConfig.LogLevel)
	}

	vpoint.space = app.NewController()
	vpoint.space.Bind(dispatcher.APP_ID, vpoint)
	vpoint.space.Bind(proxyman.APP_ID_INBOUND_MANAGER, vpoint)

	ichConfig := pConfig.InboundConfig.Settings
	ich, err := proxyrepo.CreateInboundHandler(pConfig.InboundConfig.Protocol, vpoint.space.ForContext("vpoint-default-inbound"), ichConfig)
	if err != nil {
		log.Error("Failed to create inbound connection handler: ", err)
		return nil, err
	}
	vpoint.ich = ich

	ochConfig := pConfig.OutboundConfig.Settings
	och, err := proxyrepo.CreateOutboundHandler(pConfig.OutboundConfig.Protocol, vpoint.space.ForContext("vpoint-default-outbound"), ochConfig)
	if err != nil {
		log.Error("Failed to create outbound connection handler: ", err)
		return nil, err
	}
	vpoint.och = och

	vpoint.taggedIdh = make(map[string]InboundDetourHandler)
	detours := pConfig.InboundDetours
	if len(detours) > 0 {
		vpoint.idh = make([]InboundDetourHandler, len(detours))
		for idx, detourConfig := range detours {
			allocConfig := detourConfig.Allocation
			var detourHandler InboundDetourHandler
			switch allocConfig.Strategy {
			case AllocationStrategyAlways:
				dh, err := NewInboundDetourHandlerAlways(vpoint.space.ForContext(detourConfig.Tag), detourConfig)
				if err != nil {
					log.Error("Point: Failed to create detour handler: ", err)
					return nil, ErrorBadConfiguration
				}
				detourHandler = dh
			case AllocationStrategyRandom:
				dh, err := NewInboundDetourHandlerDynamic(vpoint.space.ForContext(detourConfig.Tag), detourConfig)
				if err != nil {
					log.Error("Point: Failed to create detour handler: ", err)
					return nil, ErrorBadConfiguration
				}
				detourHandler = dh
			default:
				log.Error("Point: Unknown allocation strategy: ", allocConfig.Strategy)
				return nil, ErrorBadConfiguration
			}
			vpoint.idh[idx] = detourHandler
			if len(detourConfig.Tag) > 0 {
				vpoint.taggedIdh[detourConfig.Tag] = detourHandler
			}
		}
	}

	outboundDetours := pConfig.OutboundDetours
	if len(outboundDetours) > 0 {
		vpoint.odh = make(map[string]proxy.OutboundHandler)
		for _, detourConfig := range outboundDetours {
			detourHandler, err := proxyrepo.CreateOutboundHandler(detourConfig.Protocol, vpoint.space.ForContext(detourConfig.Tag), detourConfig.Settings)
			if err != nil {
				log.Error("Failed to create detour outbound connection handler: ", err)
				return nil, err
			}
			vpoint.odh[detourConfig.Tag] = detourHandler
		}
	}

	routerConfig := pConfig.RouterConfig
	if routerConfig != nil {
		r, err := router.CreateRouter(routerConfig.Strategy, routerConfig.Settings)
		if err != nil {
			log.Error("Failed to create router: ", err)
			return nil, ErrorBadConfiguration
		}
		vpoint.router = r
	}

	return vpoint, nil
}

func (this *Point) Close() {
	this.ich.Close()
	for _, idh := range this.idh {
		idh.Close()
	}
}

// Start starts the Point server, and return any error during the process.
// In the case of any errors, the state of the server is unpredicatable.
func (this *Point) Start() error {
	if this.port <= 0 {
		log.Error("Invalid port ", this.port)
		return ErrorBadConfiguration
	}

	err := retry.Timed(100 /* times */, 100 /* ms */).On(func() error {
		err := this.ich.Listen(this.port)
		if err != nil {
			return err
		}
		log.Warning("Point server started on port ", this.port)
		return nil
	})
	if err != nil {
		return err
	}

	for _, detourHandler := range this.idh {
		err := detourHandler.Start()
		if err != nil {
			return err
		}
	}

	return nil
}

// Dispatches a Packet to an OutboundConnection.
// The packet will be passed through the router (if configured), and then sent to an outbound
// connection with matching tag.
func (this *Point) DispatchToOutbound(context app.Context, packet v2net.Packet) ray.InboundRay {
	direct := ray.NewRay()
	dest := packet.Destination()

	dispatcher := this.och

	if this.router != nil {
		if tag, err := this.router.TakeDetour(dest); err == nil {
			if handler, found := this.odh[tag]; found {
				log.Info("Point: Taking detour [", tag, "] for [", dest, "]", tag, dest)
				dispatcher = handler
			} else {
				log.Warning("Point: Unable to find routing destination: ", tag)
			}
		}
	}

	go this.FilterPacketAndDispatch(packet, direct, dispatcher)
	return direct
}

func (this *Point) FilterPacketAndDispatch(packet v2net.Packet, link ray.OutboundRay, dispatcher proxy.OutboundHandler) {
	// Filter empty packets
	chunk := packet.Chunk()
	moreChunks := packet.MoreChunks()
	changed := false
	for chunk == nil && moreChunks {
		changed = true
		chunk, moreChunks = <-link.OutboundInput()
	}
	if chunk == nil && !moreChunks {
		log.Info("Point: No payload to dispatch, stopping dispatching now.")
		close(link.OutboundOutput())
		return
	}

	if changed {
		packet = v2net.NewPacket(packet.Destination(), chunk, moreChunks)
	}

	dispatcher.Dispatch(packet, link)
}

func (this *Point) GetHandler(context app.Context, tag string) (proxy.InboundHandler, int) {
	handler, found := this.taggedIdh[tag]
	if !found {
		log.Warning("Point: Unable to find an inbound handler with tag: ", tag)
		return nil, 0
	}
	return handler.GetConnectionHandler()
}
