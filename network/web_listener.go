package network

import (
	"github.com/15mga/kiwi"
	"github.com/15mga/kiwi/util"
	"github.com/fasthttp/websocket"
	"net/http"
)

type webOption struct {
	addr      string
	upgrader  *websocket.Upgrader
	resHeader http.Header
}

type WebOption func(option *webOption)

func WebAddr(addr string) WebOption {
	return func(option *webOption) {
		option.addr = addr
	}
}

func WebUpgrader(fn func(*websocket.Upgrader)) WebOption {
	return func(option *webOption) {
		fn(option.upgrader)
	}
}

func WebResHeader(header http.Header) WebOption {
	return func(option *webOption) {
		option.resHeader = header
	}
}

func NewWebListener(onConn func(conn *websocket.Conn), opts ...WebOption) kiwi.IListener {
	o := &webOption{
		addr:     ":7737",
		upgrader: &websocket.Upgrader{},
	}
	for _, opt := range opts {
		opt(o)
	}
	return &webListener{
		option: o,
		onConn: onConn,
	}
}

type webListener struct {
	server *http.Server
	option *webOption
	onConn func(conn *websocket.Conn)
}

func (l *webListener) Addr() string {
	return l.option.addr
}

func (l *webListener) Port() int {
	port, _ := util.ParseAddrPort(l.server.Addr)
	return port
}

func (l *webListener) handler(writer http.ResponseWriter, request *http.Request) {
	conn, e := l.option.upgrader.Upgrade(writer, request, l.option.resHeader)
	if e != nil {
		kiwi.Error3(util.EcServiceErr, e)
		return
	}
	l.onConn(conn)
}

func (l *webListener) Start() *util.Err {
	kiwi.Info("start websocket listener", util.M{
		"addr": l.option.addr,
	})
	http.HandleFunc("/", l.handler)

	go func() {
		e := http.ListenAndServe(l.option.addr, nil)
		if e != nil {
			panic(e)
		}
	}()
	return nil
}

func (l *webListener) Close() {
	l.server.Close()
}
