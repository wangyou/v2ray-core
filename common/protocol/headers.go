package protocol

import (
	v2net "github.com/v2ray/v2ray-core/common/net"
	"github.com/v2ray/v2ray-core/common/serial"
	"github.com/v2ray/v2ray-core/common/uuid"
)

type RequestCommand byte

const (
	RequestCommandTCP = RequestCommand(0x01)
	RequestCommandUDP = RequestCommand(0x02)
)

const (
	RequestOptionChunkStream = RequestOption(0x01)
)

type RequestOption byte

func (this RequestOption) IsChunkStream() bool {
	return (this & RequestOptionChunkStream) == RequestOptionChunkStream
}

type RequestHeader struct {
	Version byte
	User    *User
	Command RequestCommand
	Option  RequestOption
	Address v2net.Address
	Port    v2net.Port
}

func (this *RequestHeader) Destination() v2net.Destination {
	if this.Command == RequestCommandUDP {
		return v2net.UDPDestination(this.Address, this.Port)
	}
	return v2net.TCPDestination(this.Address, this.Port)
}

type ResponseCommand interface{}

type ResponseHeader struct {
	Command ResponseCommand
}

type CommandSwitchAccount struct {
	Host     v2net.Address
	Port     v2net.Port
	ID       *uuid.UUID
	AlterIds serial.Uint16Literal
	Level    UserLevel
	ValidMin byte
}
