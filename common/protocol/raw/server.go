package raw

import (
	"crypto/md5"
	"hash/fnv"
	"io"

	"github.com/v2ray/v2ray-core/common/alloc"
	"github.com/v2ray/v2ray-core/common/crypto"
	"github.com/v2ray/v2ray-core/common/log"
	v2net "github.com/v2ray/v2ray-core/common/net"
	"github.com/v2ray/v2ray-core/common/protocol"
	"github.com/v2ray/v2ray-core/common/serial"
	"github.com/v2ray/v2ray-core/transport"
)

type ServerSession struct {
	userValidator   protocol.UserValidator
	requestBodyKey  []byte
	requestBodyIV   []byte
	responseBodyKey []byte
	responseBodyIV  []byte
	responseHeader  byte
	responseWriter  io.Writer
}

func NewServerSession(validator protocol.UserValidator) *ServerSession {
	return &ServerSession{
		userValidator: validator,
	}
}

func (this *ServerSession) DecodeRequestHeader(reader io.Reader) (*protocol.RequestHeader, error) {
	buffer := alloc.NewSmallBuffer()
	defer buffer.Release()

	_, err := io.ReadFull(reader, buffer.Value[:protocol.IDBytesLen])
	if err != nil {
		log.Error("Raw: Failed to read request header: ", err)
		return nil, err
	}

	user, timestamp, valid := this.userValidator.Get(buffer.Value[:protocol.IDBytesLen])
	if !valid {
		return nil, protocol.ErrorInvalidUser
	}

	timestampHash := md5.New()
	timestampHash.Write(hashTimestamp(timestamp))
	iv := timestampHash.Sum(nil)
	aesStream := crypto.NewAesDecryptionStream(user.ID.CmdKey(), iv)
	decryptor := crypto.NewCryptionReader(aesStream, reader)

	nBytes, err := io.ReadFull(decryptor, buffer.Value[:41])
	if err != nil {
		log.Debug("Raw: Failed to read request header (", nBytes, " bytes): ", err)
		return nil, err
	}
	bufferLen := nBytes

	request := &protocol.RequestHeader{
		User:    user,
		Version: buffer.Value[0],
	}

	if request.Version != Version {
		log.Warning("Raw: Invalid protocol version ", request.Version)
		return nil, protocol.ErrorInvalidVersion
	}

	this.requestBodyIV = append([]byte(nil), buffer.Value[1:17]...)   // 16 bytes
	this.requestBodyKey = append([]byte(nil), buffer.Value[17:33]...) // 16 bytes
	this.responseHeader = buffer.Value[33]                            // 1 byte
	request.Option = protocol.RequestOption(buffer.Value[34])         // 1 byte + 2 bytes reserved
	request.Command = protocol.RequestCommand(buffer.Value[37])

	request.Port = v2net.PortFromBytes(buffer.Value[38:40])

	switch buffer.Value[40] {
	case AddrTypeIPv4:
		nBytes, err = io.ReadFull(decryptor, buffer.Value[41:45]) // 4 bytes
		bufferLen += 4
		if err != nil {
			log.Debug("VMess: Failed to read target IPv4 (", nBytes, " bytes): ", err)
			return nil, err
		}
		request.Address = v2net.IPAddress(buffer.Value[41:45])
	case AddrTypeIPv6:
		nBytes, err = io.ReadFull(decryptor, buffer.Value[41:57]) // 16 bytes
		bufferLen += 16
		if err != nil {
			log.Debug("VMess: Failed to read target IPv6 (", nBytes, " bytes): ", nBytes, err)
			return nil, err
		}
		request.Address = v2net.IPAddress(buffer.Value[41:57])
	case AddrTypeDomain:
		nBytes, err = io.ReadFull(decryptor, buffer.Value[41:42])
		if err != nil {
			log.Debug("VMess: Failed to read target domain (", nBytes, " bytes): ", nBytes, err)
			return nil, err
		}
		domainLength := int(buffer.Value[41])
		if domainLength == 0 {
			return nil, transport.ErrorCorruptedPacket
		}
		nBytes, err = io.ReadFull(decryptor, buffer.Value[42:42+domainLength])
		if err != nil {
			log.Debug("VMess: Failed to read target domain (", nBytes, " bytes): ", nBytes, err)
			return nil, err
		}
		bufferLen += 1 + domainLength
		domainBytes := append([]byte(nil), buffer.Value[42:42+domainLength]...)
		request.Address = v2net.DomainAddress(string(domainBytes))
	}

	nBytes, err = io.ReadFull(decryptor, buffer.Value[bufferLen:bufferLen+4])
	if err != nil {
		log.Debug("VMess: Failed to read checksum (", nBytes, " bytes): ", nBytes, err)
		return nil, err
	}

	fnv1a := fnv.New32a()
	fnv1a.Write(buffer.Value[:bufferLen])
	actualHash := fnv1a.Sum32()
	expectedHash := serial.BytesLiteral(buffer.Value[bufferLen : bufferLen+4]).Uint32Value()

	if actualHash != expectedHash {
		return nil, transport.ErrorCorruptedPacket
	}

	return request, nil
}

func (this *ServerSession) DecodeRequestBody(reader io.Reader) io.Reader {
	aesStream := crypto.NewAesDecryptionStream(this.requestBodyKey, this.requestBodyIV)
	return crypto.NewCryptionReader(aesStream, reader)
}

func (this *ServerSession) EncodeResponseHeader(header *protocol.ResponseHeader, writer io.Writer) {
	responseBodyKey := md5.Sum(this.requestBodyKey)
	responseBodyIV := md5.Sum(this.requestBodyIV)
	this.responseBodyKey = responseBodyKey[:]
	this.responseBodyIV = responseBodyIV[:]

	aesStream := crypto.NewAesEncryptionStream(this.responseBodyKey, this.responseBodyIV)
	encryptionWriter := crypto.NewCryptionWriter(aesStream, writer)
	this.responseWriter = encryptionWriter

	encryptionWriter.Write([]byte{this.responseHeader, 0x00})
	err := MarshalCommand(header.Command, encryptionWriter)
	if err != nil {
		encryptionWriter.Write([]byte{0x00, 0x00})
	}
}

func (this *ServerSession) EncodeResponseBody(writer io.Writer) io.Writer {
	return this.responseWriter
}
