package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl/app"
	"github.com/ugent-library/bbl/pgxrepo"
	"golang.org/x/sync/errgroup"
)

func init() {
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the server",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := newLogger(cmd.OutOrStdout())

		conn, err := pgxpool.New(cmd.Context(), config.PgConn)
		if err != nil {
			return err
		}
		defer conn.Close()

		repo, err := pgxrepo.New(cmd.Context(), conn)
		if err != nil {
			return err
		}

		index, err := newIndex(cmd.Context())
		if err != nil {
			return err
		}

		riverClient, err := newRiverClient(logger, conn, repo, index)
		if err != nil {
			return err
		}

		signalCtx, signalRelease := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer signalRelease()

		// ldapAdapter, err := services.NewLDAPAdapter(ctx, config)
		// if err != nil {
		// 	return err
		// }

		handler, err := app.New(&app.Config{
			Env:     config.Env,
			BaseURL: config.BaseURL,
			Logger:  logger,
			Repo:    repo,
			Index:   index,
			// UserSource:       ldapAdapter,
			// CookieSecret:     []byte(config.CookieSecret),
			// CookieHashSecret: []byte(config.CookieHashSecret),
			// AuthIssuerURL:    config.OIDC.IssuerURL,
			// AuthClientID:     config.OIDC.ClientID,
			// AuthClientSecret: config.OIDC.ClientSecret,
		})
		if err != nil {
			return err
		}

		server := &http.Server{
			Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
			WriteTimeout: time.Second * 15,
			ReadTimeout:  time.Second * 15,
			IdleTimeout:  time.Second * 60,
			Handler:      handler,
		}

		group, groupCtx := errgroup.WithContext(signalCtx)

		group.Go(func() error {
			logger.Info("server listening", "host", config.Host, "port", config.Port)

			if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			return nil
		})

		group.Go(func() error {
			<-groupCtx.Done()

			logger.Info("gracefully stopping server")

			timeoutCtx, timeoutRelease := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer timeoutRelease()

			err := server.Shutdown(timeoutCtx)
			if err == nil {
				logger.Info("gracefully stopped server")
			}
			return err
		})

		group.Go(func() error {
			if err := riverClient.Start(groupCtx); err != nil {
				return err
			}

			logger.Info("workers started")

			<-riverClient.Stopped()
			return nil
		})

		group.Go(func() error {
			<-groupCtx.Done()

			logger.Info("gracefully stopping workers")

			timeoutCtx, timeoutRelease := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer timeoutRelease()

			err := riverClient.Stop(timeoutCtx)
			if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
				return err
			}
			if err == nil {
				logger.Info("gracefully stopped workers")
				return nil
			}

			logger.Info("hard stopping workers")

			hardTimeoutCtx, hardTimeoutRelease := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer hardTimeoutRelease()

			err = riverClient.StopAndCancel(hardTimeoutCtx)
			if err != nil && errors.Is(err, context.DeadlineExceeded) {
				logger.Info("hard stopped workers")
				return nil
			}
			return err
		})

		if err := group.Wait(); err != nil {
			return err
		}

		logger.Info("stopped")

		return nil
	},
}
