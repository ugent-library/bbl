package cli

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl/app"
	"github.com/ugent-library/bbl/oidcauth"
	"golang.org/x/sync/errgroup"
)

func newStartCmd(e *env) *cobra.Command {
	host := envStrOr("BBL_HOST", "localhost")
	port := envIntOr("BBL_PORT", 3000)
	var dev bool

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			svc, err := e.services(ctx)
			if err != nil {
				return err
			}
			defer svc.Repo.Close()

			// Build auth providers: registered factories first, then built-in types.
			authProviders := make(map[string]app.AuthProvider)
			for name, factory := range e.reg.authProviderFactories {
				provider, err := factory(e.cfg)
				if err != nil {
					return fmt.Errorf("auth provider %q: %w", name, err)
				}
				authProviders[name] = provider
			}
			for name, ac := range e.cfg.AuthProviders {
				if _, ok := authProviders[name]; ok {
					continue
				}
				switch ac.Type {
				case "oidc":
					var c oidcauth.Config
					if err := ac.Config.Decode(&c); err != nil {
						return fmt.Errorf("auth provider %q: decode config: %w", name, err)
					}
					if c.RedirectURL == "" && e.cfg.RootURL != "" {
						c.RedirectURL = e.cfg.RootURL + "/backoffice/auth/callback/" + name
					}
					provider, err := oidcauth.New(ctx, c, []byte(e.cfg.HashSecret), []byte(e.cfg.Secret), e.cfg.Secure)
					if err != nil {
						return fmt.Errorf("auth provider %q: %w", name, err)
					}
					authProviders[name] = provider
				default:
					return fmt.Errorf("auth provider %q: unknown type %q", name, ac.Type)
				}
			}

			a, err := app.New(app.Config{
				Logger:     slog.Default(),
				Services:   svc,
				RootURL:    e.cfg.RootURL,
				Dev:        dev,
				Auth:       authProviders,
				HashSecret: []byte(e.cfg.HashSecret),
				Secret:     []byte(e.cfg.Secret),
				Secure:     e.cfg.Secure,
			})
			if err != nil {
				return err
			}
			addr := fmt.Sprintf("%s:%d", host, port)

			server := &http.Server{
				Addr:         addr,
				Handler:      a.Handler(),
				ReadTimeout:  15 * time.Second,
				WriteTimeout: 15 * time.Second,
				IdleTimeout:  60 * time.Second,
			}

			ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			g, ctx := errgroup.WithContext(ctx)

			g.Go(func() error {
				slog.Info("server starting", "addr", addr)
				if err := server.ListenAndServe(); err != http.ErrServerClosed {
					return err
				}
				return nil
			})

			g.Go(func() error {
				<-ctx.Done()
				stop()
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				return server.Shutdown(shutdownCtx)
			})

			// Future: g.Go(func() error { return worker.Run(ctx) })

			if err := g.Wait(); err != nil {
				return err
			}
			slog.Info("server stopped")
			return nil
		},
	}

	cmd.Flags().StringVar(&host, "host", host, "Listen host [$BBL_HOST]")
	cmd.Flags().IntVar(&port, "port", port, "Listen port [$BBL_PORT]")
	cmd.Flags().BoolVar(&dev, "dev", false, "Dev mode: serve assets from disk, no caching")

	return cmd
}
