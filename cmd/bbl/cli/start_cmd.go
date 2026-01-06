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

	"github.com/centrifugal/gocent/v3"
	"github.com/spf13/cobra"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app"
	"github.com/ugent-library/bbl/oaipmh"
	"github.com/ugent-library/bbl/oaiservice"
	"github.com/ugent-library/bbl/sru"
	"github.com/ugent-library/bbl/tasks"
	"github.com/ugent-library/catbird"
	"github.com/ugent-library/oidc"
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
		log := newLogger(cmd.OutOrStdout())

		repo, close, err := newRepo(cmd.Context())
		if err != nil {
			return err
		}
		defer close()

		store, err := newStore()
		if err != nil {
			return err
		}

		index, err := newIndex(cmd.Context())
		if err != nil {
			return err
		}

		centrifugeClient := gocent.New(gocent.Config{
			Addr: config.Centrifuge.API.URL,
			Key:  config.Centrifuge.API.Key,
		})

		authProvider, err := oidc.NewAuth(cmd.Context(), oidc.Config{
			IssuerURL:        config.OIDC.IssuerURL,
			ClientID:         config.OIDC.ClientID,
			ClientSecret:     config.OIDC.ClientSecret,
			RedirectURL:      config.BaseURL + "/backoffice/auth/callback",
			CookieInsecure:   config.Env == "development",
			CookiePrefix:     "bbl.oidc.",
			CookieHashSecret: []byte(config.HashSecret),
			CookieSecret:     []byte(config.Secret),
		})
		if err != nil {
			return err
		}

		oaiProvider, err := oaipmh.NewProvider(oaipmh.ProviderConfig{
			RepositoryName: "Ghent University Institutional Repository",
			BaseURL:        "http://localhost:3000/oai",
			AdminEmails:    []string{"nicolas.steenlant@ugent.be"},
			DeletedRecord:  "persistent",
			Granularity:    "YYYY-MM-DDThh:mm:ssZ",
			// StyleSheet:     "/oai.xsl",
			Backend: oaiservice.New(repo),
			ErrorHandler: func(err error) { // TODO
				log.Error("oai error", "error", err)
			},
		})
		if err != nil {
			return err
		}

		sruServer := &sru.Server{
			Host: config.Host,
			Port: config.Port,
			SearchProvider: func(q string, size int) ([][]byte, int, error) {
				hits, err := index.Works().Search(cmd.Context(), &bbl.SearchOpts{
					Query: q,
					Size:  size,
				})
				if err != nil {
					return nil, 0, err
				}

				recs := make([][]byte, len(hits.Hits))
				for i, hit := range hits.Hits {
					b, err := bbl.EncodeWork(hit.Rec, "oai_dc")
					if err != nil {
						return nil, 0, err
					}
					recs[i] = b
				}

				return recs, hits.Total, nil
			},
		}

		handler, err := app.NewApp(
			config.BaseURL,
			config.Env,
			log,
			[]byte(config.HashSecret),
			[]byte(config.Secret),
			repo,
			store,
			index,
			authProvider,
			oaiProvider,
			sruServer,
			config.Centrifuge.Transport.URL,
			[]byte(config.Centrifuge.HMACSecret),
		)
		if err != nil {
			return err
		}

		server := &http.Server{
			Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
			WriteTimeout: time.Second * 15,
			ReadTimeout:  time.Second * 15,
			IdleTimeout:  time.Second * 60,
			Handler:      handler.Handler(),
		}

		signalCtx, signalRelease := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer signalRelease()

		group, groupCtx := errgroup.WithContext(signalCtx)

		group.Go(func() error {
			log.Info("server listening", "host", config.Host, "port", config.Port)

			if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			return nil
		})

		group.Go(func() error {
			<-groupCtx.Done()

			log.Info("gracefully stopping server")

			timeoutCtx, timeoutRelease := context.WithTimeout(cmd.Context(), 5*time.Second)
			defer timeoutRelease()

			err := server.Shutdown(timeoutCtx)
			if err == nil {
				log.Info("gracefully stopped server")
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
					msgs, err := repo.Catbird.Read(groupCtx, bbl.OutboxQueue, n, hideFor)
					if err != nil {
						return err
					}

					for _, msg := range msgs {
						switch msg.Topic {
						case bbl.UserChangedTopic:
							var payload bbl.RecordChangedPayload
							if err := json.Unmarshal(msg.Payload, &payload); err != nil {
								return err
							}
							rec, err := repo.GetUser(groupCtx, payload.ID)
							if err != nil {
								return err
							}
							if err := index.Users().Add(groupCtx, rec); err != nil {
								return err
							}
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
							_, err = repo.Catbird.RunTask(groupCtx,
								tasks.AddRepresentationsName,
								tasks.AddRepresentationsInput{WorkID: payload.ID},
								catbird.RunTaskOpts{DeduplicationID: payload.ID},
							)
							_, err = repo.Catbird.RunTask(groupCtx,
								tasks.NotifySubscribersName,
								tasks.NotifySubscribersInput{Topic: msg.Topic, Payload: msg.Payload},
								catbird.RunTaskOpts{DeduplicationID: payload.ID},
							)
						}

						if _, err := repo.Catbird.Delete(groupCtx, bbl.OutboxQueue, msg.ID); err != nil {
							return err
						}
					}

					if len(msgs) < n {
						time.Sleep(500 * time.Millisecond)
					}
				}
			}

		})

		// Run catbird worker
		group.Go(func() error {
			log.Info("starting catbird worker")

			worker, err := repo.Catbird.NewWorker(catbird.WorkerOpts{
				Log: log,
				Tasks: []*catbird.Task{
					repo.Catbird.GCTask(),
					tasks.AddListItems(repo, index),
					tasks.AddRepresentations(repo, index),
					tasks.ChangeWorks(repo, index, log),
					tasks.ExportWorks(store, repo, index, centrifugeClient),
					tasks.ImportUserSource(repo, log),
					tasks.ImportWork(repo),
					tasks.ImportWorkSource(repo),
					tasks.NotifySubscriber(repo),
					tasks.NotifySubscribers(repo),
					tasks.ReindexOrganizations(repo, index),
					tasks.ReindexPeople(repo, index),
					tasks.ReindexProjects(repo, index),
					tasks.ReindexWorks(repo, index),
					tasks.ReindexUsers(repo, index),
				},
			})
			if err != nil {
				return fmt.Errorf("failed to create catbird worker: %w", err)
			}
			err = worker.Start(groupCtx)
			if err != nil {
				return fmt.Errorf("failed to start catbird worker: %w", err)
			}
			return nil
		})

		if err := group.Wait(); err != nil {
			log.Error("stopped with error", "error", err)
			return err
		}

		log.Info("stopped")

		return nil
	},
}
