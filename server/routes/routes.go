package routes

import (
	"context"
	"fmt"
	"net/http"

	"github.com/tender-barbarian/gniot/server/handlers"
	gocrud "github.com/tender-barbarian/go-crud"
)

type genericRepo[M gocrud.Model] interface {
	Create(ctx context.Context, model M) (int, error)
	Get(ctx context.Context, id int) (M, error)
	GetAll(ctx context.Context) ([]M, error)
	Delete(ctx context.Context, id int) error
	Update(ctx context.Context, model M, id int) error
	GetTable() string
}

func RegisterGenericRoutes[M gocrud.Model](ctx context.Context, repo genericRepo[M], mux *http.ServeMux, handlers *handlers.Handlers) *http.ServeMux {
	gocrud.RegisterCreate(fmt.Sprintf("POST /%s", repo.GetTable()), mux, repo.Create, handlers)
	gocrud.RegisterGet(fmt.Sprintf("GET /%s/{id}", repo.GetTable()), mux, repo.Get, handlers)
	gocrud.RegisterGetAll(fmt.Sprintf("GET /%s", repo.GetTable()), mux, repo.GetAll, handlers)
	gocrud.RegisterDelete(fmt.Sprintf("DELETE /%s/{id}", repo.GetTable()), mux, repo.Delete, handlers)
	gocrud.RegisterUpdate(fmt.Sprintf("POST /%s/{id}", repo.GetTable()), mux, repo.Update, handlers)

	return mux
}

func RegisterCustomRoutes(mux *http.ServeMux, h *handlers.Handlers) *http.ServeMux {
	return mux
}
