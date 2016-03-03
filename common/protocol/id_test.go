package protocol_test

import (
	"testing"

	. "github.com/v2ray/v2ray-core/common/protocol"
	"github.com/v2ray/v2ray-core/common/serial"
	"github.com/v2ray/v2ray-core/common/uuid"
	v2testing "github.com/v2ray/v2ray-core/testing"
	"github.com/v2ray/v2ray-core/testing/assert"
)

func TestCmdKey(t *testing.T) {
	v2testing.Current(t)

	id := NewID(uuid.New())
	assert.Bool(serial.BytesLiteral(id.CmdKey()).All(0)).IsFalse()
}
