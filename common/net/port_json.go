// +build json

package net

import (
	"encoding/json"
	"strings"

	"github.com/v2ray/v2ray-core/common/log"
)

func parseIntPort(data []byte) (Port, error) {
	var intPort int
	err := json.Unmarshal(data, &intPort)
	if err != nil {
		return Port(0), err
	}
	return PortFromInt(intPort)
}

func parseStringPort(data []byte) (Port, Port, error) {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return Port(0), Port(0), err
	}
	pair := strings.SplitN(s, "-", 2)
	if len(pair) == 0 {
		return Port(0), Port(0), ErrorInvalidPortRange
	}
	if len(pair) == 1 {
		port, err := PortFromString(pair[0])
		return port, port, err
	}

	fromPort, err := PortFromString(pair[0])
	if err != nil {
		return Port(0), Port(0), err
	}
	toPort, err := PortFromString(pair[1])
	if err != nil {
		return Port(0), Port(0), err
	}
	return fromPort, toPort, nil
}

// UnmarshalJSON implements encoding/json.Unmarshaler.UnmarshalJSON
func (this *PortRange) UnmarshalJSON(data []byte) error {
	port, err := parseIntPort(data)
	if err == nil {
		this.From = port
		this.To = port
		return nil
	}

	from, to, err := parseStringPort(data)
	if err == nil {
		this.From = from
		this.To = to
		if this.From > this.To {
			log.Error("Invalid port range ", this.From, " -> ", this.To)
			return ErrorInvalidPortRange
		}
		return nil
	}

	log.Error("Invalid port range: ", string(data))
	return ErrorInvalidPortRange
}
