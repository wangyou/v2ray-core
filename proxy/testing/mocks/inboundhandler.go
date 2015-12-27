package mocks

import (
	"io"
	"sync"

	"github.com/v2ray/v2ray-core/app"
	v2net "github.com/v2ray/v2ray-core/common/net"
	"github.com/v2ray/v2ray-core/proxy/common/connhandler"
)

type InboundConnectionHandler struct {
	Port       v2net.Port
	space      app.Space
	ConnInput  io.Reader
	ConnOutput io.Writer
}

func (this *InboundConnectionHandler) Listen(port v2net.Port) error {
	this.Port = port
	return nil
}

func (this *InboundConnectionHandler) Communicate(packet v2net.Packet) error {
	ray := this.space.PacketDispatcher().DispatchToOutbound(packet)

	input := ray.InboundInput()
	output := ray.InboundOutput()

	readFinish := &sync.Mutex{}
	writeFinish := &sync.Mutex{}

	readFinish.Lock()
	writeFinish.Lock()

	go func() {
		v2net.ReaderToChan(input, this.ConnInput)
		close(input)
		readFinish.Unlock()
	}()

	go func() {
		v2net.ChanToWriter(this.ConnOutput, output)
		writeFinish.Unlock()
	}()

	readFinish.Lock()
	writeFinish.Lock()
	return nil
}

func (this *InboundConnectionHandler) Create(space app.Space, config interface{}) (connhandler.InboundConnectionHandler, error) {
	this.space = space
	return this, nil
}
