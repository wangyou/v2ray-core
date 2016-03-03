package http_test

import (
	"bufio"
	"net/http"
	"strings"
	"testing"

	testdispatcher "github.com/v2ray/v2ray-core/app/dispatcher/testing"
	v2nettesting "github.com/v2ray/v2ray-core/common/net/testing"
	netassert "github.com/v2ray/v2ray-core/common/net/testing/assert"
	. "github.com/v2ray/v2ray-core/proxy/http"
	v2testing "github.com/v2ray/v2ray-core/testing"
	"github.com/v2ray/v2ray-core/testing/assert"
)

func TestHopByHopHeadersStrip(t *testing.T) {
	v2testing.Current(t)

	rawRequest := `GET /pkg/net/http/ HTTP/1.1
Host: golang.org
Connection: keep-alive,Foo, Bar
Foo: foo
Bar: bar
Proxy-Connection: keep-alive
Proxy-Authenticate: abc
User-Agent: Mozilla/5.0 (Macintosh; U; Intel Mac OS X; de-de) AppleWebKit/523.10.3 (KHTML, like Gecko) Version/3.0.4 Safari/523.10
Accept-Encoding: gzip
Accept-Charset: ISO-8859-1,UTF-8;q=0.7,*;q=0.7
Cache-Control: no-cache
Accept-Language: de,en;q=0.7,en-us;q=0.3

`
	b := bufio.NewReader(strings.NewReader(rawRequest))
	req, err := http.ReadRequest(b)
	assert.Error(err).IsNil()
	assert.StringLiteral(req.Header.Get("Foo")).Equals("foo")
	assert.StringLiteral(req.Header.Get("Bar")).Equals("bar")
	assert.StringLiteral(req.Header.Get("Connection")).Equals("keep-alive,Foo, Bar")
	assert.StringLiteral(req.Header.Get("Proxy-Connection")).Equals("keep-alive")
	assert.StringLiteral(req.Header.Get("Proxy-Authenticate")).Equals("abc")

	StripHopByHopHeaders(req)
	assert.StringLiteral(req.Header.Get("Connection")).Equals("close")
	assert.StringLiteral(req.Header.Get("Foo")).Equals("")
	assert.StringLiteral(req.Header.Get("Bar")).Equals("")
	assert.StringLiteral(req.Header.Get("Proxy-Connection")).Equals("")
	assert.StringLiteral(req.Header.Get("Proxy-Authenticate")).Equals("")
}

func TestNormalGetRequest(t *testing.T) {
	v2testing.Current(t)

	testPacketDispatcher := testdispatcher.NewTestPacketDispatcher(nil)

	httpProxy := NewHttpProxyServer(&Config{}, testPacketDispatcher)
	defer httpProxy.Close()

	port := v2nettesting.PickPort()
	err := httpProxy.Listen(port)
	assert.Error(err).IsNil()
	netassert.Port(port).Equals(httpProxy.Port())

	httpClient := &http.Client{}
	resp, err := httpClient.Get("http://127.0.0.1:" + port.String() + "/")
	assert.Error(err).IsNil()
	assert.Int(resp.StatusCode).Equals(400)
}
