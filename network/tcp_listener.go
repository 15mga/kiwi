package network

import (
	"github.com/15mga/kiwi"
	"net"

	"github.com/15mga/kiwi/util"
)

func NewTcpListener(addr string, onConn func(conn net.Conn)) kiwi.IListener {
	return &tcpListener{
		addr:   addr,
		onConn: onConn,
	}
}

type tcpListener struct {
	addr     string
	listener *net.TCPListener
	onConn   func(conn net.Conn)
}

func (l *tcpListener) Addr() string {
	return l.addr
}

func (l *tcpListener) Port() int {
	port, _ := util.ParseAddrPort(l.listener.Addr().String())
	return port
}

func (l *tcpListener) Start() *util.Err {
	addr, err := net.ResolveTCPAddr("tcp", l.addr)
	if err != nil {
		return util.NewErr(util.EcListenErr, util.M{
			"addr":  l.addr,
			"error": err.Error(),
		})
	}

	l.listener, err = net.ListenTCP("tcp", addr)
	if err != nil {
		return util.NewErr(util.EcListenErr, util.M{
			"addr":  l.addr,
			"error": err.Error(),
		})
	}

	go func() {
		for {
			conn, err := l.listener.AcceptTCP()
			if err != nil {
				kiwi.Error(util.WrapErr(util.EcAcceptErr, err))
				return
			}
			l.onConn(conn)
		}
	}()
	return nil
}

func (l *tcpListener) Close() {
	if l.listener == nil {
		return
	}
	l.listener.Close()
}
