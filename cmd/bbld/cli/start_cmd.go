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
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app"
	"github.com/ugent-library/bbl/pgadapter"
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
		conn, err := pgxpool.New(cmd.Context(), config.PgConn)
		if err != nil {
			return err
		}
		defer conn.Close()
		repo := bbl.NewRepo(pgadapter.New(conn))

		logger := newLogger(cmd.OutOrStdout())

		signalCtx, signalRelease := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer signalRelease()

		// if err := biblio.LoadWorkProfiles(config.WorkProfiles); err != nil {
		// 	return err
		// }

		// pgAdapter, err := services.NewPostgresAdapter(ctx, config)
		// if err != nil {
		// 	return err
		// }
		// defer pgAdapter.Cleanup()

		// index, err := services.NewOpenSearchIndex(ctx, config)
		// if err != nil {
		// 	return err
		// }

		// ldapAdapter, err := services.NewLDAPAdapter(ctx, config)
		// if err != nil {
		// 	return err
		// }

		handler, err := app.New(&app.Config{
			Env:     config.Env,
			BaseURL: config.BaseURL,
			Logger:  logger,
			Repo:    repo,
			// Queue:            pgAdapter.Queue(),
			// Index:            index,
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

		// group.Go(func() error {
		// 	return jobs.Funnel(groupCtx, pgAdapter.Queue(), "work_indexer", biblio.WorkTopic, pgAdapter.Works().Get, index.IndexWork)
		// })

		// group.Go(func() error {
		// 	return jobs.Funnel(groupCtx, pgAdapter.Queue(), "project_indexer", biblio.ProjectTopic, pgAdapter.Projects().Get, index.IndexProject)
		// })

		// group.Go(func() error {
		// 	return jobs.WorkRepresentations(groupCtx, pgAdapter.Queue(), pgAdapter.WorkRepresentations(), pgAdapter.Works(), services.WorkEncoders())
		// })

		group.Go(func() error {
			logger.Info("server listening", "host", config.Host, "port", config.Port)
			if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			return nil
		})

		group.Go(func() error {
			<-groupCtx.Done()
			logger.Info("gracefully stopping")
			timeoutCtx, timeoutRelease := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer timeoutRelease()
			return server.Shutdown(timeoutCtx)
		})

		if err := group.Wait(); err != nil {
			return err
		}

		logger.Info("stopped")

		return nil
	},
}
