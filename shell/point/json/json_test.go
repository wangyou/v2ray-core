package json_test

import (
	"path/filepath"
	"testing"

	netassert "github.com/v2ray/v2ray-core/common/net/testing/assert"
	_ "github.com/v2ray/v2ray-core/proxy/dokodemo/json"
	_ "github.com/v2ray/v2ray-core/proxy/freedom/json"
	_ "github.com/v2ray/v2ray-core/proxy/socks/json"
	_ "github.com/v2ray/v2ray-core/proxy/vmess/inbound/json"
	_ "github.com/v2ray/v2ray-core/proxy/vmess/outbound/json"
	"github.com/v2ray/v2ray-core/shell/point/json"

	v2testing "github.com/v2ray/v2ray-core/testing"
	"github.com/v2ray/v2ray-core/testing/assert"
)

func TestClientSampleConfig(t *testing.T) {
	v2testing.Current(t)

	// TODO: fix for Windows
	baseDir := "$GOPATH/src/github.com/v2ray/v2ray-core/release/config"

	pointConfig, err := json.LoadConfig(filepath.Join(baseDir, "vpoint_socks_vmess.json"))
	assert.Error(err).IsNil()

	netassert.Port(pointConfig.Port()).IsValid()
	assert.Pointer(pointConfig.InboundConfig()).IsNotNil()
	assert.Pointer(pointConfig.OutboundConfig()).IsNotNil()

	assert.StringLiteral(pointConfig.InboundConfig().Protocol()).Equals("socks")
	assert.Pointer(pointConfig.InboundConfig().Settings()).IsNotNil()

	assert.StringLiteral(pointConfig.OutboundConfig().Protocol()).Equals("vmess")
	assert.Pointer(pointConfig.OutboundConfig().Settings()).IsNotNil()
}

func TestServerSampleConfig(t *testing.T) {
	v2testing.Current(t)

	// TODO: fix for Windows
	baseDir := "$GOPATH/src/github.com/v2ray/v2ray-core/release/config"

	pointConfig, err := json.LoadConfig(filepath.Join(baseDir, "vpoint_vmess_freedom.json"))
	assert.Error(err).IsNil()

	assert.Uint16(pointConfig.Port().Value()).Positive()
	assert.Pointer(pointConfig.InboundConfig()).IsNotNil()
	assert.Pointer(pointConfig.OutboundConfig()).IsNotNil()

	assert.StringLiteral(pointConfig.InboundConfig().Protocol()).Equals("vmess")
	assert.Pointer(pointConfig.InboundConfig().Settings()).IsNotNil()

	assert.StringLiteral(pointConfig.OutboundConfig().Protocol()).Equals("freedom")
	assert.Pointer(pointConfig.OutboundConfig().Settings()).IsNotNil()
}
