// +build json

package http

import (
	"encoding/json"

	v2net "github.com/v2ray/v2ray-core/common/net"
	"github.com/v2ray/v2ray-core/proxy/internal/config"
)

func (this *Config) UnmarshalJSON(data []byte) error {
	type JsonConfig struct {
		Hosts []v2net.AddressJson `json:"ownHosts"`
	}
	jsonConfig := new(JsonConfig)
	if err := json.Unmarshal(data, jsonConfig); err != nil {
		return err
	}
	this.OwnHosts = make([]v2net.Address, len(jsonConfig.Hosts), len(jsonConfig.Hosts)+1)
	for idx, host := range jsonConfig.Hosts {
		this.OwnHosts[idx] = host.Address
	}

	v2rayHost := v2net.DomainAddress("local.v2ray.com")
	if !this.IsOwnHost(v2rayHost) {
		this.OwnHosts = append(this.OwnHosts, v2rayHost)
	}

	return nil
}

func init() {
	config.RegisterInboundConfig("http",
		func(data []byte) (interface{}, error) {
			rawConfig := new(Config)
			err := json.Unmarshal(data, rawConfig)
			return rawConfig, err
		})
}
