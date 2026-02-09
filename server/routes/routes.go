package routes

import (
	"context"
	"fmt"
	"net/http"

	"github.com/tender-barbarian/gniotek/repository"
	"github.com/tender-barbarian/gniotek/server/handlers"
	gocrud "github.com/tender-barbarian/go-crud"
)

func RegisterGenericRoutes[M gocrud.Model](ctx context.Context, mux *http.ServeMux, eh *handlers.ErrorHandler, repo repository.GenericRepo[M]) *http.ServeMux {
	gocrud.RegisterCreate(fmt.Sprintf("POST /%s", repo.GetTable()), mux, repo.Create, eh)
	gocrud.RegisterGet(fmt.Sprintf("GET /%s/{id}", repo.GetTable()), mux, repo.Get, eh)
	gocrud.RegisterGetAll(fmt.Sprintf("GET /%s", repo.GetTable()), mux, repo.GetAll, eh)
	gocrud.RegisterDelete(fmt.Sprintf("DELETE /%s/{id}", repo.GetTable()), mux, repo.Delete, eh)
	gocrud.RegisterUpdate(fmt.Sprintf("POST /%s/{id}", repo.GetTable()), mux, repo.Update, eh)

	return mux
}

func RegisterCustomRoutes(mux *http.ServeMux, h *handlers.CustomHandlers) *http.ServeMux {
	mux.HandleFunc("POST /execute", h.Execute)
	return mux
}
