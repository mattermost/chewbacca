package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/mattermost/chewbacca/internal/api"
	"github.com/mattermost/chewbacca/internal/github"
	"github.com/mattermost/chewbacca/model"

	"github.com/gorilla/mux"
	logrus "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var instanceID string

func init() {
	instanceID = model.NewID()

	serverCmd.PersistentFlags().String("listen", ":8075", "The interface and port on which to listen.")
	serverCmd.PersistentFlags().String("github-token", "", "The GitHub token to the bot be able to interact.")
	serverCmd.PersistentFlags().String("github-secret", "", "The GitHub secret key to use to validate the request from github.")
	serverCmd.PersistentFlags().Bool("debug", false, "Whether to output debug logs.")
	serverCmd.PersistentFlags().Bool("machine-readable-logs", false, "Output the logs in machine readable format.")
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run Chewbacca server.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		debug, _ := command.Flags().GetBool("debug")
		if debug {
			logger.SetLevel(logrus.DebugLevel)
		}

		machineLogs, _ := command.Flags().GetBool("machine-readable-logs")
		if machineLogs {
			logger.SetFormatter(&logrus.JSONFormatter{})
		}

		logger := logger.WithField("instance", instanceID)

		logger.WithFields(logrus.Fields{
			"debug": debug,
		}).Info("Starting Chewbacca Server")

		gitHubToken, _ := command.Flags().GetString("github-token")
		gitHubSecret, _ := command.Flags().GetString("github-secret")

		gitHubClient := github.NewGitHubConfig(gitHubToken, gitHubSecret, logger)

		router := mux.NewRouter()

		api.Register(router, &api.Context{
			GitHub: gitHubClient,
			Logger: logger,
		})

		listen, _ := command.Flags().GetString("listen")
		srv := &http.Server{
			Addr:           listen,
			Handler:        router,
			ReadTimeout:    180 * time.Second,
			WriteTimeout:   180 * time.Second,
			IdleTimeout:    time.Second * 180,
			MaxHeaderBytes: 1 << 20,
			ErrorLog:       log.New(&logrusWriter{logger}, "", 0),
		}

		go func() {
			logger.WithField("addr", srv.Addr).Info("Listening")
			err := srv.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				logger.WithError(err).Error("Failed to listen and serve")
			}
		}()

		c := make(chan os.Signal, 1)
		// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
		// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
		signal.Notify(c, os.Interrupt)

		// Block until we receive our signal.
		<-c
		logger.Info("Shutting down")

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		srv.Shutdown(ctx)

		return nil
	},
}
