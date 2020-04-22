package api

import (
	"net/http"

	"github.com/gorilla/mux"
)

// Register registers the API endpoints on the given router.
func Register(rootRouter *mux.Router, context *Context) {

	apiRouter := rootRouter.PathPrefix("/api").Subrouter()
	rootRouter.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	initGitHubWebhook(apiRouter, context)
}
