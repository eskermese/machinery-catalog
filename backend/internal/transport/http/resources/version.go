package resources

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	httpx "github.com/eskermese/machinery-catalog/backend/pkg/httpx"
)

const APIVersion = "v1"

// VersionResponse - ответ на запрос версии.
type VersionResponse struct {
	API     string `json:"api"`
	Version string `json:"version"`
}

// VersionResource - структура содержащая версию API и приложения.
type VersionResource struct {
	Version string
}

// Routes возвращает роутер для раздачи версии API.
func (vr VersionResource) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", vr.Get)

	return r
}

// Get возвращает версию API.
func (vr VersionResource) Get(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, VersionResponse{
		API:     APIVersion,
		Version: vr.Version,
	})
}
