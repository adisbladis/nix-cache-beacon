//go:build linux
package main

import (
	"context"
	"fmt"

	"github.com/vishvananda/netlink"
)

func watchInterfaces(ctx context.Context, callback func()) error {
	updates := make(chan netlink.AddrUpdate)
	done := make(chan struct{})

	if err := netlink.AddrSubscribe(updates, done); err != nil {
		return fmt.Errorf("netlink addr subscribe failed: %w", err)
	}
	defer close(done)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update := <-updates:
			if !update.NewAddr {
				continue
			}
			ip := update.LinkAddress.IP
			if !ip.IsLoopback() && ip.IsGlobalUnicast() {
				callback()
			}
		}
	}
}
