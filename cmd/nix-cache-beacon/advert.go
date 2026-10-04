package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/betamos/zeroconf"
	"github.com/gofrs/uuid/v5"
	"github.com/urfave/cli/v2"

	"github.com/adisbladis/nix-cache-beacon/internal/constants"
)

func runAdvert(cliCtx *cli.Context) error {
	hostname := cliCtx.String("hostname")
	if hostname == "" {
		localHostname, err := os.Hostname()
		if err != nil {
			return err
		}
		hostname = localHostname
	}

	// Qualify unqualified hostnames with the mDNS domain (e.g. "nixos" -> "nixos.local").
	// Otherwise the advertised SRV target is not resolvable.
	if !strings.Contains(hostname, ".") {
		hostname += "." + constants.ServiceType.Domain
	}

	port := cliCtx.Int("port")

	id, err := uuid.NewV4()
	if err != nil {
		return err
	}
	name := id.String()

	svc := zeroconf.Service{
		Type:     constants.ServiceType,
		Name:     name,
		Port:     uint16(port),
		Hostname: hostname,
	}

	server, err := zeroconf.New().Publish(&svc).Open()
	if err != nil {
		return err
	}
	defer server.Close()

	slog.Info("started", "id", name, "topic", constants.MDNS_SERVICE, "hostname", hostname, "port", port)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	go func() {
		if err := watchInterfaces(ctx, func() {
			slog.Debug("change in network interface, reload advert")
			server.Reload()
		}); err != nil {
			slog.Error("error watching interfaces", "error", err)
		}
	}()

	<-ctx.Done()

	return nil
}
