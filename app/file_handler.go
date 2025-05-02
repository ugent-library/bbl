package app

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/s3store"
	"github.com/ugent-library/bbl/ctx"
)

type FileHandler struct {
	store *s3store.Store
}

func NewFileHandler(store *s3store.Store) *FileHandler {
	return &FileHandler{
		store: store,
	}
}

func (h *FileHandler) AddRoutes(router *mux.Router, appCtx *ctx.Ctx[*AppCtx]) {
	router.Handle("/files/upload_url", appCtx.Bind(h.CreateUploadURL)).Methods("POST").Name("create_file_upload_url")
}

func (h *FileHandler) CreateUploadURL(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	w.Header().Set("Content-Type", "application/json")
	req := struct {
		Name        string `json:"name"`
		ContentType string `json:"content_type"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}
	fileID := bbl.NewID()
	presignedURL, err := h.store.NewUploadURL(r.Context(), fileID, 15*time.Minute)
	if err != nil {
		return err
	}
	res := struct {
		URL string `json:"url"`
	}{
		URL: presignedURL,
	}
	if err := json.NewEncoder(w).Encode(&res); err != nil {
		return err
	}
	return nil
}
