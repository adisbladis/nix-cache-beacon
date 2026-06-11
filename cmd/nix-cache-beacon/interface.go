//go:build !linux
package main

import (
	"context"
	"fmt"
	"net"

	"golang.org/x/net/route"
	"golang.org/x/sys/unix"
)

func watchInterfaces(ctx context.Context, onUp func()) error {
	fd, err := unix.Socket(unix.AF_ROUTE, unix.SOCK_RAW, unix.AF_UNSPEC)
	if err != nil {
		return fmt.Errorf("routing socket failed: %w", err)
	}
	defer unix.Close(fd)

	go func() {
		<-ctx.Done()
		unix.Close(fd)
	}()

	buf := make([]byte, 4096)
	for {
		n, err := unix.Read(fd, buf)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("routing socket read: %w", err)
		}

		msgs, err := route.ParseRIB(route.RIBTypeRoute, buf[:n])
		if err != nil {
			continue
		}

		for _, msg := range msgs {
			ifam, ok := msg.(*route.InterfaceAddrMessage)
			if !ok || ifam.Type != unix.RTM_NEWADDR {
				continue
			}
			for _, addr := range ifam.Addrs {
				if addr == nil {
					continue
				}

				var ip net.IP
				switch a := addr.(type) {
				case *route.Inet4Addr:
					ip = net.IP(a.IP[:])
				case *route.Inet6Addr:
					ip = net.IP(a.IP[:])
				}

				if ip != nil && !ip.IsLoopback() && ip.IsGlobalUnicast() {
					onUp()
				}
			}
		}
	}
}
