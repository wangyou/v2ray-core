package socks

import (
	"net"

	"github.com/v2ray/v2ray-core/common/alloc"
	"github.com/v2ray/v2ray-core/common/log"
	v2net "github.com/v2ray/v2ray-core/common/net"
	"github.com/v2ray/v2ray-core/proxy/socks/protocol"
)

var udpAddress v2net.Destination

func (this *SocksServer) ListenUDP(port v2net.Port) error {
	addr := &net.UDPAddr{
		IP:   net.IP{0, 0, 0, 0},
		Port: int(port),
		Zone: "",
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Error("Socks failed to listen UDP on port %d: %v", port, err)
		return err
	}
	udpAddress = v2net.UDPDestination(v2net.IPAddress(this.config.IP()), port)

	go this.AcceptPackets(conn)
	return nil
}

func (this *SocksServer) getUDPAddr() v2net.Destination {
	return udpAddress
}

func (this *SocksServer) AcceptPackets(conn *net.UDPConn) error {
	for {
		buffer := alloc.NewBuffer()
		nBytes, addr, err := conn.ReadFromUDP(buffer.Value)
		if err != nil {
			log.Error("Socks failed to read UDP packets: %v", err)
			buffer.Release()
			continue
		}
		log.Info("Client UDP connection from %v", addr)
		request, err := protocol.ReadUDPRequest(buffer.Value[:nBytes])
		buffer.Release()
		if err != nil {
			log.Error("Socks failed to parse UDP request: %v", err)
			continue
		}
		if request.Data == nil || request.Data.Len() == 0 {
			continue
		}
		if request.Fragment != 0 {
			log.Warning("Dropping fragmented UDP packets.")
			// TODO handle fragments
			request.Data.Release()
			continue
		}

		udpPacket := v2net.NewPacket(request.Destination(), request.Data, false)
		log.Info("Send packet to %s with %d bytes", udpPacket.Destination().String(), request.Data.Len())
		go this.handlePacket(conn, udpPacket, addr, request.Address, request.Port)
	}
}

func (this *SocksServer) handlePacket(conn *net.UDPConn, packet v2net.Packet, clientAddr *net.UDPAddr, targetAddr v2net.Address, port v2net.Port) {
	ray := this.space.PacketDispatcher().DispatchToOutbound(packet)
	close(ray.InboundInput())

	for data := range ray.InboundOutput() {
		response := &protocol.Socks5UDPRequest{
			Fragment: 0,
			Address:  targetAddr,
			Port:     port,
			Data:     data,
		}
		log.Info("Writing back UDP response with %d bytes from %s to %s", data.Len(), targetAddr.String(), clientAddr.String())

		udpMessage := alloc.NewSmallBuffer().Clear()
		response.Write(udpMessage)

		nBytes, err := conn.WriteToUDP(udpMessage.Value, clientAddr)
		udpMessage.Release()
		response.Data.Release()
		if err != nil {
			log.Error("Socks failed to write UDP message (%d bytes) to %s: %v", nBytes, clientAddr.String(), err)
		}
	}
}
