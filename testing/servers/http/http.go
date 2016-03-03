package tcp

import (
	"net/http"

	v2net "github.com/v2ray/v2ray-core/common/net"
)

type Server struct {
	Port        v2net.Port
	PathHandler map[string]http.HandlerFunc
	accepting   bool
}

func (server *Server) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/" {
		resp.Header().Set("Content-Type", "text/plain; charset=utf-8")
		resp.WriteHeader(http.StatusOK)
		resp.Write([]byte("Home"))
		return
	}

	handler, found := server.PathHandler[req.URL.Path]
	if found {
		handler(resp, req)
	}
}

func (server *Server) Start() (v2net.Destination, error) {
	go http.ListenAndServe(":"+server.Port.String(), server)
	return v2net.TCPDestination(v2net.IPAddress([]byte{127, 0, 0, 1}), v2net.Port(server.Port)), nil
}

func (this *Server) Close() {
	this.accepting = false
}
