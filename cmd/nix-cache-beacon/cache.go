package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/betamos/zeroconf"
	"github.com/nix-community/go-nix/pkg/narinfo"
	"github.com/urfave/cli/v2"

	"github.com/adisbladis/nix-cache-beacon/internal/config"
	"github.com/adisbladis/nix-cache-beacon/internal/constants"
	"github.com/adisbladis/nix-cache-beacon/internal/index"
	"github.com/cenkalti/backoff/v5"
)

func makeHandler(cfg *config.Config, cacheIndex *index.CacheIndex, client *http.Client) http.Handler {
	cacheInfo := []byte(cfg.CacheInfo.String())

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Handle cache info
		if r.URL.Path == "/nix-cache-info" {
			if _, err := w.Write(cacheInfo); err != nil {
				slog.Warn("error writing nix-cache-info", "err", err)
			}
			return
		}

		// Handle narinfo
		if r.URL.Path[0] == '/' && strings.HasSuffix(r.URL.Path, ".narinfo") {
			ninfo, ok := findNarInfo(r.Context(), cfg, cacheIndex, r.URL.RequestURI(), client)
			if !ok {
				http.NotFound(w, r)
				return
			}

			if _, err := w.Write([]byte(ninfo.String())); err != nil {
				slog.Warn("error writing narinfo response", "URL", r.URL.RequestURI(), "err", err)
			}

			return
		}

		// Everything else unhandled
		http.NotFound(w, r)
	})
}

func findNarInfo(ctx context.Context, cfg *config.Config, cacheIndex *index.CacheIndex, path string, client *http.Client) (*narinfo.NarInfo, bool) {
	type result struct {
		narinfo *narinfo.NarInfo
		err     error
	}

	groupCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for caches := range cacheIndex.Iter() {
		ch := make(chan result)

		var wg sync.WaitGroup
		wg.Add(len(caches))

		priority := 0
		if len(caches) > 0 {
			priority = caches[0].Priority
		}

		slog.Info("retreiving narinfo", "path", path, "caches", len(caches), "priority", priority)

		// Race priority group
		for _, cache := range caches {

			go func() {
				defer wg.Done()

				ninfo, err := cache.GetNarInfo(groupCtx, path, client)
				if err != nil {
					// Not found isn't a transport error
					if err == index.NotFoundError {
						return
					}

					// Evict cache on network failures
					if _, ok := errors.AsType[net.Error](err); ok {
						slog.Info("evicting", "URL", cache.URL)
						go cacheIndex.Evict(cache.URL, client)
					}

					slog.Error("got error for narinfo", "path", path, "cache", cache.URL, "error", err)
					ch <- result{err: err}
					return
				}

				// Check if signed by any known keys, or is a fixed output
				if !strings.HasPrefix(ninfo.CA, "fixed:") {
					fingerprint := ninfo.Fingerprint()

					signed := false
					for _, sig := range ninfo.Signatures {
						pub, ok := cfg.Keys[sig.Name]
						if !ok {
							continue
						}

						if pub.Verify(fingerprint, sig) {
							signed = true
						}
					}

					if !signed {
						slog.Warn("could not find any valid signatures for narinfo", "path", path, "cache", cache.URL)
						ch <- result{err: errors.New("could not find any valid signatures")}
						return
					}
				}

				ch <- result{narinfo: ninfo}
			}()
		}

		go func() {
			wg.Wait()
			close(ch)
		}()

		for r := range ch {
			if r.err == nil {
				return r.narinfo, true
			}
		}
	}

	return nil, false
}

func readConfig(configPath string) (*config.Config, error) {
	cfg := config.NewConfig()
	if configPath != "" {
		slog.Info("Using config file", "path", configPath)

		f, err := os.Open(configPath)
		if err != nil {
			log.Fatalf("failed to open config %q: %v", configPath, err)
		}
		defer f.Close()

		data, err := io.ReadAll(f)
		if err != nil {
			return nil, err
		}

		err = config.Unmarshal(data, cfg)
		if err != nil {
			log.Fatalf("failed to read config: %v", err)
		}
	} else {
		slog.Info("no config file specified")
	}
	return cfg, nil
}

func runCache(cliCtx *cli.Context) (err error) {
	cfg, err := readConfig(cliCtx.Path("config"))
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: cfg.RequestTimeout,
		// Allow unencrypted HTTP2
		Transport: &http.Transport{
			Protocols: func() *http.Protocols {
				p := &http.Protocols{}
				p.SetHTTP1(true)
				p.SetUnencryptedHTTP2(true)
				return p
			}(),
		},
	}

	// Exit on abort
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	// In-memory binary cache index
	cacheIndex := index.NewCacheIndex()

	// Create HTTP server, with h2c support
	srv := &http.Server{Handler: makeHandler(cfg, cacheIndex, client)}
	srv.Protocols = new(http.Protocols)
	srv.Protocols.SetHTTP1(true)
	srv.Protocols.SetUnencryptedHTTP2(true)

	// Listen to servers
	listenAddrs := cliCtx.StringSlice("listen")
	{
		for _, addr := range listenAddrs {
			l, err := net.Listen("tcp", addr)
			if err != nil {
				log.Fatalf("failed to listen on %s: %v", addr, err)
			}

			go func() {
				slog.Info("nix-cache-beacon listening", "address", l.Addr())
				if err := srv.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Fatalf("server error on %s: %v", l.Addr(), err)
				}
			}()
		}
	}

	// Browse zeroconf
	{
		client, err := zeroconf.New().
			Browse(func(event zeroconf.Event) {
				if !event.Service.Type.Equal(constants.ServiceType) {
					return
				}

				go func() {
					cacheURL := fmt.Sprintf("http://%s", net.JoinHostPort(event.Hostname, fmt.Sprintf("%d", event.Port)))

					switch event.Op {
					case zeroconf.OpRemoved:
						cacheIndex.Remove(cacheURL)
						slog.Info("removing", "URL", cacheURL)
					case zeroconf.OpAdded, zeroconf.OpUpdated:
						cache, err := cacheIndex.Get(cacheURL)
						if err != nil && err == index.ErrNotFound {
							cache = index.NewBinaryCache(cacheURL, -1)
						} else if err != nil {
							panic(err)
						}

						operation := func() (struct{}, error) {
							slog.Info("retrying", "URL", cacheURL)

							cacheInfo, err := cache.GetCacheInfo(ctx, client)
							if err != nil {
								slog.Warn("error retreiving nix-cache-info", "URL", cacheURL, "err", err)
								return struct{}{}, nil
							}

							cache.Priority = cacheInfo.Priority
							slog.Info("adding", "URL", cache.URL, "priority", cache.Priority)
							cacheIndex.Add(cache)
							return struct{}{}, nil
						}

						_, err = backoff.Retry(ctx, operation, backoff.WithBackOff(backoff.NewExponentialBackOff()))
						if err != nil {
							panic(err)
						}
					}
				}()
			}, constants.ServiceType).
			Open()
		if err != nil {
			return err
		}
		defer client.Close() // Don't forget to close, to notify others that we're going away
	}

	// Wait for shutdown signal.
	{
		<-ctx.Done()

		slog.Info("shutting down", "timeout", shutdownTimeout)

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Warn("shutdown error", "err", err)
			return errors.New("shutdown completed with errors")
		}
	}

	return nil
}
