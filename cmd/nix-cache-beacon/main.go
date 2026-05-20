package main

import (
	"log"
	"os"
	"time"

	"github.com/urfave/cli/v2"
)

const (
	shutdownTimeout = 30 * time.Second
)

func main() {
	app := &cli.App{
		Name:  "nix-cache-beacon",
		Usage: "Nix binary cache discovery",
		Commands: []*cli.Command{
			{
				Name:  "cache",
				Usage: "Run the cache server",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:    "listen",
						Aliases: []string{"l"},
						Usage:   "Address(es) to listen on (repeatable, e.g. --listen 0.0.0.0:8080)",
						Required: true,
					},
					&cli.PathFlag{
						Name:    "config",
						Aliases: []string{"c"},
						Usage:   "Path to a local config file",
					},
				},
				Action: runCache,
			},
			{
				Name:  "advert",
				Usage: "Advertise a cache on the network",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:     "port",
						Aliases:  []string{"p"},
						Usage:    "Port number to advertise",
						Required: true,
					},
				},
				Action: runAdvert,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
