package socks

import (
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/v2ray/v2ray-core/app"
	"github.com/v2ray/v2ray-core/common/alloc"
	"github.com/v2ray/v2ray-core/common/log"
	v2net "github.com/v2ray/v2ray-core/common/net"
	"github.com/v2ray/v2ray-core/common/retry"
	proxyerrors "github.com/v2ray/v2ray-core/proxy/common/errors"
	"github.com/v2ray/v2ray-core/proxy/socks/protocol"
)

var (
	UnsupportedSocksCommand = errors.New("Unsupported socks command.")
	UnsupportedAuthMethod   = errors.New("Unsupported auth method.")
)

// SocksServer is a SOCKS 5 proxy server
type SocksServer struct {
	accepting bool
	space     app.Space
	config    Config
}

func NewSocksServer(space app.Space, config Config) *SocksServer {
	return &SocksServer{
		space:  space,
		config: config,
	}
}

func (this *SocksServer) Listen(port v2net.Port) error {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   []byte{0, 0, 0, 0},
		Port: int(port),
		Zone: "",
	})
	if err != nil {
		log.Error("Socks failed to listen on port %d: %v", port, err)
		return err
	}
	this.accepting = true
	go this.AcceptConnections(listener)
	if this.config.UDPEnabled() {
		this.ListenUDP(port)
	}
	return nil
}

func (this *SocksServer) AcceptConnections(listener *net.TCPListener) {
	for this.accepting {
		retry.Timed(100 /* times */, 100 /* ms */).On(func() error {
			connection, err := listener.AcceptTCP()
			if err != nil {
				log.Error("Socks failed to accept new connection %v", err)
				return err
			}
			go this.HandleConnection(connection)
			return nil
		})

	}
}

func (this *SocksServer) HandleConnection(connection *net.TCPConn) error {
	defer connection.Close()

	reader := v2net.NewTimeOutReader(120, connection)

	auth, auth4, err := protocol.ReadAuthentication(reader)
	if err != nil && err != protocol.Socks4Downgrade {
		log.Error("Socks failed to read authentication: %v", err)
		return err
	}

	if err != nil && err == protocol.Socks4Downgrade {
		return this.handleSocks4(reader, connection, auth4)
	} else {
		return this.handleSocks5(reader, connection, auth)
	}
}

func (this *SocksServer) handleSocks5(reader *v2net.TimeOutReader, writer io.Writer, auth protocol.Socks5AuthenticationRequest) error {
	expectedAuthMethod := protocol.AuthNotRequired
	if this.config.IsPassword() {
		expectedAuthMethod = protocol.AuthUserPass
	}

	if !auth.HasAuthMethod(expectedAuthMethod) {
		authResponse := protocol.NewAuthenticationResponse(protocol.AuthNoMatchingMethod)
		err := protocol.WriteAuthentication(writer, authResponse)
		if err != nil {
			log.Error("Socks failed to write authentication: %v", err)
			return err
		}
		log.Warning("Socks client doesn't support allowed any auth methods.")
		return UnsupportedAuthMethod
	}

	authResponse := protocol.NewAuthenticationResponse(expectedAuthMethod)
	err := protocol.WriteAuthentication(writer, authResponse)
	if err != nil {
		log.Error("Socks failed to write authentication: %v", err)
		return err
	}
	if this.config.IsPassword() {
		upRequest, err := protocol.ReadUserPassRequest(reader)
		if err != nil {
			log.Error("Socks failed to read username and password: %v", err)
			return err
		}
		status := byte(0)
		if !this.config.HasAccount(upRequest.Username(), upRequest.Password()) {
			status = byte(0xFF)
		}
		upResponse := protocol.NewSocks5UserPassResponse(status)
		err = protocol.WriteUserPassResponse(writer, upResponse)
		if err != nil {
			log.Error("Socks failed to write user pass response: %v", err)
			return err
		}
		if status != byte(0) {
			log.Warning("Invalid user account: %s", upRequest.AuthDetail())
			return proxyerrors.InvalidAuthentication
		}
	}

	request, err := protocol.ReadRequest(reader)
	if err != nil {
		log.Error("Socks failed to read request: %v", err)
		return err
	}

	if request.Command == protocol.CmdUdpAssociate && this.config.UDPEnabled() {
		return this.handleUDP(reader, writer)
	}

	if request.Command == protocol.CmdBind || request.Command == protocol.CmdUdpAssociate {
		response := protocol.NewSocks5Response()
		response.Error = protocol.ErrorCommandNotSupported
		response.Port = v2net.Port(0)
		response.SetIPv4([]byte{0, 0, 0, 0})

		responseBuffer := alloc.NewSmallBuffer().Clear()
		response.Write(responseBuffer)
		_, err = writer.Write(responseBuffer.Value)
		responseBuffer.Release()
		if err != nil {
			log.Error("Socks failed to write response: %v", err)
			return err
		}
		log.Warning("Unsupported socks command %d", request.Command)
		return UnsupportedSocksCommand
	}

	response := protocol.NewSocks5Response()
	response.Error = protocol.ErrorSuccess

	// Some SOCKS software requires a value other than dest. Let's fake one:
	response.Port = v2net.Port(1717)
	response.SetIPv4([]byte{0, 0, 0, 0})

	responseBuffer := alloc.NewSmallBuffer().Clear()
	response.Write(responseBuffer)
	_, err = writer.Write(responseBuffer.Value)
	responseBuffer.Release()
	if err != nil {
		log.Error("Socks failed to write response: %v", err)
		return err
	}

	dest := request.Destination()
	log.Info("TCP Connect request to %s", dest.String())

	packet := v2net.NewPacket(dest, nil, true)
	this.transport(reader, writer, packet)
	return nil
}

func (this *SocksServer) handleUDP(reader *v2net.TimeOutReader, writer io.Writer) error {
	response := protocol.NewSocks5Response()
	response.Error = protocol.ErrorSuccess

	udpAddr := this.getUDPAddr()

	response.Port = udpAddr.Port()
	switch {
	case udpAddr.Address().IsIPv4():
		response.SetIPv4(udpAddr.Address().IP())
	case udpAddr.Address().IsIPv6():
		response.SetIPv6(udpAddr.Address().IP())
	case udpAddr.Address().IsDomain():
		response.SetDomain(udpAddr.Address().Domain())
	}

	responseBuffer := alloc.NewSmallBuffer().Clear()
	response.Write(responseBuffer)
	_, err := writer.Write(responseBuffer.Value)
	responseBuffer.Release()

	if err != nil {
		log.Error("Socks failed to write response: %v", err)
		return err
	}

	reader.SetTimeOut(300)      /* 5 minutes */
	v2net.ReadFrom(reader, nil) // Just in case of anything left in the socket
	// The TCP connection closes after this method returns. We need to wait until
	// the client closes it.
	// TODO: get notified from UDP part
	<-time.After(5 * time.Minute)

	return nil
}

func (this *SocksServer) handleSocks4(reader io.Reader, writer io.Writer, auth protocol.Socks4AuthenticationRequest) error {
	result := protocol.Socks4RequestGranted
	if auth.Command == protocol.CmdBind {
		result = protocol.Socks4RequestRejected
	}
	socks4Response := protocol.NewSocks4AuthenticationResponse(result, auth.Port, auth.IP[:])

	responseBuffer := alloc.NewSmallBuffer().Clear()
	socks4Response.Write(responseBuffer)
	writer.Write(responseBuffer.Value)
	responseBuffer.Release()

	if result == protocol.Socks4RequestRejected {
		log.Warning("Unsupported socks 4 command %d", auth.Command)
		return UnsupportedSocksCommand
	}

	dest := v2net.TCPDestination(v2net.IPAddress(auth.IP[:]), auth.Port)
	packet := v2net.NewPacket(dest, nil, true)
	this.transport(reader, writer, packet)
	return nil
}

func (this *SocksServer) transport(reader io.Reader, writer io.Writer, firstPacket v2net.Packet) {
	ray := this.space.PacketDispatcher().DispatchToOutbound(firstPacket)
	input := ray.InboundInput()
	output := ray.InboundOutput()

	var inputFinish, outputFinish sync.Mutex
	inputFinish.Lock()
	outputFinish.Lock()

	go func() {
		v2net.ReaderToChan(input, reader)
		inputFinish.Unlock()
		close(input)
	}()

	go func() {
		v2net.ChanToWriter(writer, output)
		outputFinish.Unlock()
	}()
	outputFinish.Lock()
}
