package netutil

import (
	"net"
	"time"

	"github.com/demskie/hyperionfive/arguments"
	"github.com/demskie/logging"

	"github.com/gorilla/websocket"
)

// GetLocalTCP ...
func GetLocalTCP(args *arguments.Args, log *logging.Logger) (tcpAddr net.TCPAddr) {
	const fName = "(GetLocalTCP)"
	if args.LocalInterface != "" {
		iface, err := net.InterfaceByName(args.LocalInterface)
		if log.CheckError(err) == false {
			addrs, anotherError := iface.Addrs()
			if log.CheckError(anotherError) == false {
				tcpAddr = net.TCPAddr{
					IP: addrs[0].(*net.IPNet).IP,
				}
			}
		}
	}
	if args.LocalAddress != "" {
		tcpAddr = net.TCPAddr{
			IP: net.ParseIP(args.LocalAddress),
		}
	} else {
		ifaces, err := net.Interfaces()
		if log.CheckError(err) == false {
			for _, iface := range ifaces {
				addrs, err := iface.Addrs()
				if log.CheckError(err) {
					continue
				}
				for _, addr := range addrs {
					if log.CheckError(err) {
						continue
					}
					tcpAddr = net.TCPAddr{
						IP: addr.(*net.IPNet).IP,
					}
					return tcpAddr
				}
			}
		}
	}
	return tcpAddr
}

// GetLocalAddress ...
func GetLocalAddress(args *arguments.Args, log *logging.Logger) string {
	return GetLocalTCP(args, log).IP.String()
}

// GetWebsocketDialer ...
func GetWebsocketDialer(args *arguments.Args, log *logging.Logger) (dialer *websocket.Dialer) {
	tcpAddr := GetLocalTCP(args, log)
	netDialer := &net.Dialer{
		LocalAddr: &tcpAddr,
		Deadline:  time.Now().Add(time.Second * 30),
	}
	dialer = &websocket.Dialer{
		NetDial: netDialer.Dial,
	}
	return dialer
}
