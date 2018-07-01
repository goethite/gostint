package doc

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
)

type Doc struct {
	Path string `json:"path"`
	Desc string `json:"desc"`
}

func Routes() *chi.Mux {
	router := chi.NewRouter()

	router.Get("/", GetAllDocs)
	return router
}

func GetAllDocs(w http.ResponseWriter, req *http.Request) {
	docs := []Doc{
		{
			Path: "/doc",
			Desc: "API Path Documentation",
		},
	}
	render.JSON(w, req, docs)
}

//
