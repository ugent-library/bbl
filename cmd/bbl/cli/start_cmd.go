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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app"
	"github.com/ugent-library/bbl/app/ctx"
	"github.com/ugent-library/bbl/catbird"
	"github.com/ugent-library/bbl/pgxrepo"
	"github.com/ugent-library/bbl/workflows"
	"golang.org/x/sync/errgroup"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
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

		store, err := newStore()
		if err != nil {
			return err
		}

		index, err := newIndex(cmd.Context())
		if err != nil {
			return err
		}

		hub := catbird.NewHub(conn, catbird.HubOpts{})

		hatchetClient, err := hatchet.NewClient()
		if err != nil {
			return err
		}

		addWorkRepresentationsTask := workflows.AddWorkRepresentations(hatchetClient, repo, index)
		exportWorksTask := workflows.ExportWorks(hatchetClient, store, index, hub)
		importUserSourceTask := workflows.ImportUserSource(hatchetClient, repo)
		importWorkSourceTask := workflows.ImportWorkSource(hatchetClient, repo)
		importWorkTask := workflows.ImportWork(hatchetClient, repo)
		reindexOrganizationsTask := workflows.ReindexOrganizations(hatchetClient, repo, index)
		reindexPeopleTask := workflows.ReindexPeople(hatchetClient, repo, index)
		tongaGCTask := workflows.TongaGC(hatchetClient, repo.Tonga)

		signalCtx, signalRelease := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer signalRelease()

		handler, err := app.New(&ctx.Config{
			Env:              config.Env,
			BaseURL:          config.BaseURL,
			Logger:           logger,
			Repo:             repo,
			Index:            index,
			Store:            store,
			Hub:              hub,
			Secret:           []byte(config.Secret),
			HashSecret:       []byte(config.HashSecret),
			AuthIssuerURL:    config.OIDC.IssuerURL,
			AuthClientID:     config.OIDC.ClientID,
			AuthClientSecret: config.OIDC.ClientSecret,
			ExportWorksTask:  exportWorksTask, // TOOD how best to pass tasks to handlers?
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
			logger.Info("started message hub")
			return hub.Start(groupCtx)
		})

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

			timeoutCtx, timeoutRelease := context.WithTimeout(cmd.Context(), 5*time.Second)
			defer timeoutRelease()

			hub.Shutdown() // TODO not graceful
			logger.Info("stopped message hub")

			err := server.Shutdown(timeoutCtx)
			if err == nil {
				logger.Info("gracefully stopped server")
				return nil
			}
			return err
		})

		// Run outbox reader
		// TODO split off command?
		// TODO use long polling / cdc
		// TODO max attempts / dead letter box
		// TODO log instead of returning where appropriate
		// TODO check if latest rev?
		group.Go(func() error {
			n := 100
			hideFor := 10 * time.Second

			for {
				select {
				case <-groupCtx.Done():
					return nil
				default:
					msgs, err := repo.Tonga.Read(groupCtx, bbl.OutboxQueue, n, hideFor)
					if err != nil {
						return err
					}

					for _, msg := range msgs {
						switch msg.Topic {
						case bbl.OrganizationChangedTopic:
							var payload bbl.RecordChangedPayload
							if err := json.Unmarshal(msg.Payload, &payload); err != nil {
								return err
							}
							rec, err := repo.GetOrganization(groupCtx, payload.ID)
							if err != nil {
								return err
							}
							if err := index.Organizations().Add(groupCtx, rec); err != nil {
								return err
							}
						case bbl.PersonChangedTopic:
							var payload bbl.RecordChangedPayload
							if err := json.Unmarshal(msg.Payload, &payload); err != nil {
								return err
							}
							rec, err := repo.GetPerson(groupCtx, payload.ID)
							if err != nil {
								return err
							}
							if err := index.People().Add(groupCtx, rec); err != nil {
								return err
							}
						case bbl.ProjectChangedTopic:
							var payload bbl.RecordChangedPayload
							if err := json.Unmarshal(msg.Payload, &payload); err != nil {
								return err
							}
							rec, err := repo.GetProject(groupCtx, payload.ID)
							if err != nil {
								return err
							}
							if err := index.Projects().Add(groupCtx, rec); err != nil {
								return err
							}
						case bbl.WorkChangedTopic:
							var payload bbl.RecordChangedPayload
							if err := json.Unmarshal(msg.Payload, &payload); err != nil {
								return err
							}
							rec, err := repo.GetWork(groupCtx, payload.ID)
							if err != nil {
								return err
							}
							if err := index.Works().Add(groupCtx, rec); err != nil {
								return err
							}
						}

						if err := hatchetClient.Events().Push(groupCtx, msg.Topic, msg.Payload); err != nil {
							return err
						}

						if _, err := repo.Tonga.Delete(groupCtx, bbl.OutboxQueue, msg.ID); err != nil {
							return err
						}
					}

					if len(msgs) < n {
						time.Sleep(500 * time.Millisecond)
					}
				}
			}

		})

		// Run Hatchet worker
		group.Go(func() error {
			logger.Info("starting hatchet worker")
			worker, err := hatchetClient.NewWorker("worker", hatchet.WithWorkflows(
				addWorkRepresentationsTask,
				exportWorksTask,
				importUserSourceTask,
				importWorkSourceTask,
				importWorkTask,
				reindexOrganizationsTask,
				reindexPeopleTask,
				tongaGCTask,
			))
			if err != nil {
				return fmt.Errorf("failed to create hatchet worker: %w", err)
			}
			err = worker.StartBlocking(groupCtx)
			if err != nil {
				return fmt.Errorf("failed to start hatchet worker: %w", err)
			}
			return nil
		})

		if err := group.Wait(); err != nil {
			return err
		}

		logger.Info("stopped")

		return nil
	},
}
