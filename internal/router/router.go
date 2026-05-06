package router

import (
	"bank-service/internal/handler"
	"net/http"

	"github.com/gorilla/mux"
)

func NewRouter() http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/health", handler.Health).Methods(http.MethodGet)
	return r
}
