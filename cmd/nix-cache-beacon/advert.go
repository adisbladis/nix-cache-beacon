package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/betamos/zeroconf"
	"github.com/gofrs/uuid/v5"
	"github.com/urfave/cli/v2"

	"github.com/adisbladis/nix-cache-beacon/internal/constants"
)

func runAdvert(ctx *cli.Context) error {
	hostname := ctx.String("hostname")
	if hostname == "" {
		localHostname, err := os.Hostname()
		if err != nil {
			return err
		}
		hostname = localHostname
	}
	port := ctx.Int("port")

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

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	return nil
}
