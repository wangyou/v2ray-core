package scenarios

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	v2net "github.com/v2ray/v2ray-core/common/net"
	v2testing "github.com/v2ray/v2ray-core/testing"
	"github.com/v2ray/v2ray-core/testing/assert"
	v2http "github.com/v2ray/v2ray-core/testing/servers/http"
)

func TestHttpProxy(t *testing.T) {
	v2testing.Current(t)

	httpServer := &v2http.Server{
		Port:        v2net.Port(50042),
		PathHandler: make(map[string]http.HandlerFunc),
	}
	_, err := httpServer.Start()
	assert.Error(err).IsNil()
	defer httpServer.Close()

	assert.Error(InitializeServerSetOnce("test_5")).IsNil()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse("http://127.0.0.1:50040/")
		},
	}

	client := &http.Client{
		Transport: transport,
	}

	resp, err := client.Get("http://127.0.0.1:50042/")
	assert.Error(err).IsNil()
	assert.Int(resp.StatusCode).Equals(200)

	content, err := ioutil.ReadAll(resp.Body)
	assert.Error(err).IsNil()
	assert.StringLiteral(string(content)).Equals("Home")

	CloseAllServers()
}
