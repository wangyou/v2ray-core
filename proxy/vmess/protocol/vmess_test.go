package protocol

import (
	"bytes"
	"crypto/rand"
	"testing"

	v2net "github.com/v2ray/v2ray-core/common/net"
	"github.com/v2ray/v2ray-core/proxy/vmess/config"
	"github.com/v2ray/v2ray-core/proxy/vmess/protocol/user"
	"github.com/v2ray/v2ray-core/testing/mocks"
	"github.com/v2ray/v2ray-core/testing/unit"
)

type TestUser struct {
	id *config.ID
}

func (u *TestUser) ID() *config.ID {
	return u.id
}

func TestVMessSerialization(t *testing.T) {
	assert := unit.Assert(t)

	userId, err := config.NewID("2b2966ac-16aa-4fbf-8d81-c5f172a3da51")
	if err != nil {
		t.Fatal(err)
	}

	userSet := mocks.MockUserSet{[]*config.ID{}, make(map[string]int), make(map[string]int64)}
	userSet.AddUser(&TestUser{userId})

	request := new(VMessRequest)
	request.Version = byte(0x01)
	request.UserId = *userId

	randBytes := make([]byte, 36)
	_, err = rand.Read(randBytes)
	assert.Error(err).IsNil()
	request.RequestIV = randBytes[:16]
	request.RequestKey = randBytes[16:32]
	request.ResponseHeader = randBytes[32:]

	request.Command = byte(0x01)
	request.Address = v2net.DomainAddress("v2ray.com", 80)

	mockTime := int64(1823730)

	buffer, err := request.ToBytes(user.NewTimeHash(user.HMACHash{}), func(base int64, delta int) int64 { return mockTime }, nil)
	if err != nil {
		t.Fatal(err)
	}

	userSet.UserHashes[string(buffer.Value[:16])] = 0
	userSet.Timestamps[string(buffer.Value[:16])] = mockTime

	requestReader := NewVMessRequestReader(&userSet)
	actualRequest, err := requestReader.Read(bytes.NewReader(buffer.Value))
	if err != nil {
		t.Fatal(err)
	}

	assert.Byte(actualRequest.Version).Named("Version").Equals(byte(0x01))
	assert.String(actualRequest.UserId.String).Named("UserId").Equals(request.UserId.String)
	assert.Bytes(actualRequest.RequestIV).Named("RequestIV").Equals(request.RequestIV[:])
	assert.Bytes(actualRequest.RequestKey).Named("RequestKey").Equals(request.RequestKey[:])
	assert.Bytes(actualRequest.ResponseHeader).Named("ResponseHeader").Equals(request.ResponseHeader[:])
	assert.Byte(actualRequest.Command).Named("Command").Equals(request.Command)
	assert.String(actualRequest.Address.String()).Named("Address").Equals(request.Address.String())
}

func BenchmarkVMessRequestWriting(b *testing.B) {
	userId, _ := config.NewID("2b2966ac-16aa-4fbf-8d81-c5f172a3da51")
	userSet := mocks.MockUserSet{[]*config.ID{}, make(map[string]int), make(map[string]int64)}
	userSet.AddUser(&TestUser{userId})

	request := new(VMessRequest)
	request.Version = byte(0x01)
	request.UserId = *userId

	randBytes := make([]byte, 36)
	rand.Read(randBytes)
	request.RequestIV = randBytes[:16]
	request.RequestKey = randBytes[16:32]
	request.ResponseHeader = randBytes[32:]

	request.Command = byte(0x01)
	request.Address = v2net.DomainAddress("v2ray.com", 80)

	for i := 0; i < b.N; i++ {
		request.ToBytes(user.NewTimeHash(user.HMACHash{}), user.GenerateRandomInt64InRange, nil)
	}
}
