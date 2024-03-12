package network

import (
	"github.com/15mga/kiwi"
	"net"

	"github.com/15mga/kiwi/util"
	"github.com/xtaci/kcp-go/v5"
)

func NewUdpListener(addr string, onConn func(conn net.Conn)) kiwi.IListener {
	return &udpListener{
		addr:   addr,
		onConn: onConn,
	}
}

type udpListener struct {
	addr     string
	onConn   func(conn net.Conn)
	listener *kcp.Listener
}

func (l *udpListener) Addr() string {
	return l.addr
}

func (l *udpListener) Port() int {
	port, _ := util.ParseAddrPort(l.listener.Addr().String())
	return port
}

func (l *udpListener) Start() *util.Err {
	kiwi.Info("start udp listener", util.M{
		"addr": l.addr,
	})
	listener, err := kcp.Listen(l.addr)
	if err != nil {
		return util.NewErr(util.EcListenErr, util.M{
			"addr":  l.addr,
			"error": err.Error(),
		})
	}

	l.listener = listener.(*kcp.Listener)
	go func() {
		for {
			conn, err := l.listener.AcceptKCP()
			if err != nil {
				kiwi.Error2(util.EcAcceptErr, util.M{
					"error": err.Error(),
				})
				return
			}

			l.onConn(conn)
		}
	}()
	return nil
}

func (l *udpListener) Close() {
	if l.listener == nil {
		return
	}
	l.listener.Close()
}
