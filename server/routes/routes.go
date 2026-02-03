package routes

import (
	"context"
	"fmt"
	"net/http"

	"github.com/tender-barbarian/gniot/repository"
	"github.com/tender-barbarian/gniot/server/handlers"
	gocrud "github.com/tender-barbarian/go-crud"
)

func RegisterGenericRoutes[M gocrud.Model](ctx context.Context, mux *http.ServeMux, h *handlers.CustomHandlers, repo repository.GenericRepo[M]) *http.ServeMux {
	gocrud.RegisterCreate(fmt.Sprintf("POST /%s", repo.GetTable()), mux, repo.Create, h)
	gocrud.RegisterGet(fmt.Sprintf("GET /%s/{id}", repo.GetTable()), mux, repo.Get, h)
	gocrud.RegisterGetAll(fmt.Sprintf("GET /%s", repo.GetTable()), mux, repo.GetAll, h)
	gocrud.RegisterDelete(fmt.Sprintf("DELETE /%s/{id}", repo.GetTable()), mux, repo.Delete, h)
	gocrud.RegisterUpdate(fmt.Sprintf("POST /%s/{id}", repo.GetTable()), mux, repo.Update, h)

	return mux
}

func RegisterGenericRoutesWithDB[M gocrud.Model](ctx context.Context, mux *http.ServeMux, h *handlers.CustomHandlers, repo repository.GenericRepo[M], db gocrud.DBQuerier) *http.ServeMux {
	gocrud.RegisterCreateWithDB(fmt.Sprintf("POST /%s", repo.GetTable()), mux, repo.Create, db, h)
	gocrud.RegisterGet(fmt.Sprintf("GET /%s/{id}", repo.GetTable()), mux, repo.Get, h)
	gocrud.RegisterGetAll(fmt.Sprintf("GET /%s", repo.GetTable()), mux, repo.GetAll, h)
	gocrud.RegisterDelete(fmt.Sprintf("DELETE /%s/{id}", repo.GetTable()), mux, repo.Delete, h)
	gocrud.RegisterUpdateWithDB(fmt.Sprintf("POST /%s/{id}", repo.GetTable()), mux, repo.Update, db, h)

	return mux
}

func RegisterCustomRoutes(mux *http.ServeMux, h *handlers.CustomHandlers) *http.ServeMux {
	mux.HandleFunc("POST /execute", h.Execute)
	return mux
}
