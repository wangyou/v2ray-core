package inbound

import (
	"github.com/v2ray/v2ray-core/common/log"
	"github.com/v2ray/v2ray-core/common/protocol"
	"github.com/v2ray/v2ray-core/common/serial"
)

func (this *VMessInboundHandler) generateCommand(request *protocol.RequestHeader) protocol.ResponseCommand {
	if this.features != nil && this.features.Detour != nil {

		tag := this.features.Detour.ToTag
		if this.inboundHandlerManager != nil {
			handler, availableMin := this.inboundHandlerManager.GetHandler(tag)
			inboundHandler, ok := handler.(*VMessInboundHandler)
			if ok {
				if availableMin > 255 {
					availableMin = 255
				}

				log.Info("VMessIn: Pick detour handler for port ", inboundHandler.Port(), " for ", availableMin, " minutes.")
				user := inboundHandler.GetUser(request.User.Email)
				return &protocol.CommandSwitchAccount{
					Port:     inboundHandler.Port(),
					ID:       user.ID.UUID(),
					AlterIds: serial.Uint16Literal(len(user.AlterIDs)),
					Level:    user.Level,
					ValidMin: byte(availableMin),
				}
			}
		}
	}

	return nil
}
