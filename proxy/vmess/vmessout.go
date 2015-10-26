package vmess

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	mrand "math/rand"
	"net"
	"sync"

	"github.com/v2ray/v2ray-core/common/alloc"
	v2io "github.com/v2ray/v2ray-core/common/io"
	"github.com/v2ray/v2ray-core/common/log"
	v2net "github.com/v2ray/v2ray-core/common/net"
	"github.com/v2ray/v2ray-core/proxy"
	"github.com/v2ray/v2ray-core/proxy/vmess/config"
	"github.com/v2ray/v2ray-core/proxy/vmess/protocol"
	"github.com/v2ray/v2ray-core/proxy/vmess/protocol/user"
	"github.com/v2ray/v2ray-core/transport/ray"
)

type VMessOutboundHandler struct {
	vNextList    []*config.OutboundTarget
	vNextListUDP []*config.OutboundTarget
}

func NewVMessOutboundHandler(vNextList, vNextListUDP []*config.OutboundTarget) *VMessOutboundHandler {
	return &VMessOutboundHandler{
		vNextList:    vNextList,
		vNextListUDP: vNextListUDP,
	}
}

func pickVNext(serverList []*config.OutboundTarget) (v2net.Destination, config.User) {
	vNextLen := len(serverList)
	if vNextLen == 0 {
		panic("VMessOut: Zero vNext is configured.")
	}
	vNextIndex := 0
	if vNextLen > 1 {
		vNextIndex = mrand.Intn(vNextLen)
	}

	vNext := serverList[vNextIndex]
	vNextUserLen := len(vNext.Accounts)
	if vNextUserLen == 0 {
		panic("VMessOut: Zero User account.")
	}
	vNextUserIndex := 0
	if vNextUserLen > 1 {
		vNextUserIndex = mrand.Intn(vNextUserLen)
	}
	vNextUser := vNext.Accounts[vNextUserIndex]
	return vNext.Destination, vNextUser
}

func (handler *VMessOutboundHandler) Dispatch(firstPacket v2net.Packet, ray ray.OutboundRay) error {
	vNextList := handler.vNextList
	if firstPacket.Destination().IsUDP() {
		vNextList = handler.vNextListUDP
	}
	vNextAddress, vNextUser := pickVNext(vNextList)

	command := protocol.CmdTCP
	if firstPacket.Destination().IsUDP() {
		command = protocol.CmdUDP
	}
	request := &protocol.VMessRequest{
		Version: protocol.Version,
		UserId:  *vNextUser.ID(),
		Command: command,
		Address: firstPacket.Destination().Address(),
	}

	buffer := alloc.NewSmallBuffer()
	defer buffer.Release()
	v2net.ReadAllBytes(rand.Reader, buffer.Value[:36]) // 16 + 16 + 4
	request.RequestIV = buffer.Value[:16]
	request.RequestKey = buffer.Value[16:32]
	request.ResponseHeader = buffer.Value[32:36]

	return startCommunicate(request, vNextAddress, ray, firstPacket)
}

func startCommunicate(request *protocol.VMessRequest, dest v2net.Destination, ray ray.OutboundRay, firstPacket v2net.Packet) error {
	conn, err := net.Dial(dest.Network(), dest.Address().String())
	if err != nil {
		log.Error("Failed to open %s: %v", dest.String(), err)
		if ray != nil {
			close(ray.OutboundOutput())
		}
		return err
	}
	log.Info("VMessOut: Tunneling request to %s via %s", request.Address.String(), dest.String())

	defer conn.Close()

	input := ray.OutboundInput()
	output := ray.OutboundOutput()
	var requestFinish, responseFinish sync.Mutex
	requestFinish.Lock()
	responseFinish.Lock()

	go handleRequest(conn, request, firstPacket, input, &requestFinish)
	go handleResponse(conn, request, output, &responseFinish, dest.IsUDP())

	requestFinish.Lock()
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.CloseWrite()
	}
	responseFinish.Lock()
	return nil
}

func handleRequest(conn net.Conn, request *protocol.VMessRequest, firstPacket v2net.Packet, input <-chan *alloc.Buffer, finish *sync.Mutex) {
	defer finish.Unlock()
	encryptRequestWriter, err := v2io.NewAesEncryptWriter(request.RequestKey[:], request.RequestIV[:], conn)
	if err != nil {
		log.Error("VMessOut: Failed to create encrypt writer: %v", err)
		return
	}

	buffer := alloc.NewBuffer().Clear()
	buffer, err = request.ToBytes(user.NewTimeHash(user.HMACHash{}), user.GenerateRandomInt64InRange, buffer)
	if err != nil {
		log.Error("VMessOut: Failed to serialize VMess request: %v", err)
		return
	}

	// Send first packet of payload together with request, in favor of small requests.
	firstChunk := firstPacket.Chunk()
	moreChunks := firstPacket.MoreChunks()

	if firstChunk == nil && moreChunks {
		firstChunk, moreChunks = <-input
	}

	if firstChunk != nil {
		encryptRequestWriter.Crypt(firstChunk.Value)
		buffer.Append(firstChunk.Value)
		firstChunk.Release()

		_, err = conn.Write(buffer.Value)
		buffer.Release()
		if err != nil {
			log.Error("VMessOut: Failed to write VMess request: %v", err)
			return
		}
	}

	if moreChunks {
		v2net.ChanToWriter(encryptRequestWriter, input)
	}
	return
}

func handleResponse(conn net.Conn, request *protocol.VMessRequest, output chan<- *alloc.Buffer, finish *sync.Mutex, isUDP bool) {
	defer finish.Unlock()
	defer close(output)
	responseKey := md5.Sum(request.RequestKey[:])
	responseIV := md5.Sum(request.RequestIV[:])

	decryptResponseReader, err := v2io.NewAesDecryptReader(responseKey[:], responseIV[:], conn)
	if err != nil {
		log.Error("VMessOut: Failed to create decrypt reader: %v", err)
		return
	}

	buffer, err := v2net.ReadFrom(decryptResponseReader, nil)
	if err != nil {
		log.Error("VMessOut: Failed to read VMess response (%d bytes): %v", buffer.Len(), err)
		return
	}
	if buffer.Len() < 4 || !bytes.Equal(buffer.Value[:4], request.ResponseHeader[:]) {
		log.Warning("VMessOut: unexepcted response header. The connection is probably hijacked.")
		return
	}
	log.Info("VMessOut received %d bytes from %s", buffer.Len()-4, conn.RemoteAddr().String())

	buffer.SliceFrom(4)
	output <- buffer

	if !isUDP {
		v2net.ReaderToChan(output, decryptResponseReader)
	}

	return
}

type VMessOutboundHandlerFactory struct {
}

func (factory *VMessOutboundHandlerFactory) Create(rawConfig interface{}) (proxy.OutboundConnectionHandler, error) {
	vOutConfig := rawConfig.(config.Outbound)
	servers := make([]*config.OutboundTarget, 0, 16)
	udpServers := make([]*config.OutboundTarget, 0, 16)
	for _, target := range vOutConfig.Targets() {
		if target.Destination.IsTCP() {
			servers = append(servers, target)
		}
		if target.Destination.IsUDP() {
			udpServers = append(udpServers, target)
		}
	}
	return NewVMessOutboundHandler(servers, udpServers), nil
}

func init() {
	proxy.RegisterOutboundConnectionHandlerFactory("vmess", &VMessOutboundHandlerFactory{})
}
