package util

import (
	"net"
	"strconv"
	"strings"
)

func GetLocalIp() (string, *Err) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", WrapErr(EcAddrErr, err)
	}
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok &&
			!n.IP.IsLoopback() &&
			n.IP.To4() != nil {
			return n.IP.String(), nil
		}
	}
	return "", WrapErr(EcAddrErr, err)
}

func IsLocalIp(ip string) bool {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok &&
			!n.IP.IsLoopback() &&
			n.IP.To4() != nil &&
			n.IP.String() == ip {
			return true
		}
	}
	return false
}

func CheckLocalIp(ip string) (string, *Err) {
	if ip == "" {
		return GetLocalIp()
	}
	if IsLocalIp(ip) {
		return ip, nil
	}
	return "", NewErr(EcAddrErr, M{
		"error": "not local ip",
	})
}

func ParseAddrPort(addr string) (int, *Err) {
	idx := strings.LastIndex(addr, ":")
	if idx < 0 {
		return 0, NewErr(EcParamsErr, M{
			"addr": addr,
		})
	}
	port, e := strconv.Atoi(addr[idx+1:])
	if e != nil {
		return 0, NewErr(EcParamsErr, M{
			"addr":  addr,
			"error": e,
		})
	}
	return port, nil
}

func GetFreePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}
