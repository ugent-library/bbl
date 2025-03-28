package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl/app"
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
		repo, close, err := NewRepo(cmd.Context())
		if err != nil {
			return err
		}
		defer close()

		index, err := NewIndex(cmd.Context())
		if err != nil {
			return err
		}

		logger := NewLogger(cmd.OutOrStdout())

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

		// person indexer
		group.Go(func() error {
			var err error
			for msg := range repo.Listen(groupCtx, "person_index", "person", 10*time.Second) {
				var id string
				if err = json.Unmarshal(msg.Body, &id); err != nil {
					logger.Error("person_index", "err", err)
					continue
				}
				rec, err := repo.GetPerson(groupCtx, id)
				if err != nil {
					logger.Error("person_index", "err", err)
					continue
				}
				if err = index.People().Add(groupCtx, rec); err != nil {
					logger.Error("person_index", "err", err)
				}
				if err = repo.Ack(groupCtx, msg); err != nil {
					logger.Error("person_index", "err", err)
				}
			}
			return err
		})

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
