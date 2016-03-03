package hub

import (
	"net"
	"time"

	"github.com/v2ray/v2ray-core/common/log"
	v2net "github.com/v2ray/v2ray-core/common/net"
)

type TCPConn struct {
	conn     *net.TCPConn
	listener *TCPHub
	dirty    bool
}

func (this *TCPConn) Read(b []byte) (int, error) {
	return this.conn.Read(b)
}

func (this *TCPConn) Write(b []byte) (int, error) {
	return this.conn.Write(b)
}

func (this *TCPConn) Close() error {
	return this.conn.Close()
}

func (this *TCPConn) Release() {
	if this.dirty {
		this.Close()
		return
	}
	this.listener.recycle(this.conn)
}

func (this *TCPConn) LocalAddr() net.Addr {
	return this.conn.LocalAddr()
}

func (this *TCPConn) RemoteAddr() net.Addr {
	return this.conn.RemoteAddr()
}

func (this *TCPConn) SetDeadline(t time.Time) error {
	return this.conn.SetDeadline(t)
}

func (this *TCPConn) SetReadDeadline(t time.Time) error {
	return this.conn.SetReadDeadline(t)
}

func (this *TCPConn) SetWriteDeadline(t time.Time) error {
	return this.conn.SetWriteDeadline(t)
}

func (this *TCPConn) CloseRead() error {
	return this.conn.CloseRead()
}

func (this *TCPConn) CloseWrite() error {
	return this.conn.CloseWrite()
}

type TCPHub struct {
	listener     *net.TCPListener
	connCallback func(*TCPConn)
	accepting    bool
}

func ListenTCP(port v2net.Port, callback func(*TCPConn)) (*TCPHub, error) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   []byte{0, 0, 0, 0},
		Port: int(port),
		Zone: "",
	})
	if err != nil {
		return nil, err
	}
	tcpListener := &TCPHub{
		listener:     listener,
		connCallback: callback,
	}
	go tcpListener.start()
	return tcpListener, nil
}

func (this *TCPHub) Close() {
	this.accepting = false
	this.listener.Close()
}

func (this *TCPHub) start() {
	this.accepting = true
	for this.accepting {
		conn, err := this.listener.AcceptTCP()
		if err != nil {
			if this.accepting {
				log.Warning("Listener: Failed to accept new TCP connection: ", err)
			}
			continue
		}
		go this.connCallback(&TCPConn{
			conn:     conn,
			listener: this,
		})
	}
}

func (this *TCPHub) recycle(conn *net.TCPConn) {

}
