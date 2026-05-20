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
	domain := ctx.String("domain")
	port := ctx.Int("port")

	id, err := uuid.NewV4()
	if err != nil {
		return err
	}
	name := id.String()

	svc := zeroconf.NewService(constants.ServiceType, name, uint16(port))

	server, err := zeroconf.New().Publish(svc).Open()
	if err != nil {
		return err
	}
	defer server.Close()

	slog.Info("started", "id", name, "topic", constants.MDNS_SERVICE, "domain", domain, "port", port)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	return nil
}
