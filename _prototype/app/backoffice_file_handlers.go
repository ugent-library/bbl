package app

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ugent-library/bbl"
)

func (app *App) createFileUploadURL(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	w.Header().Set("Content-Type", "application/json")

	req := struct {
		Name        string `json:"name"`
		ContentType string `json:"content_type"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}

	objectID := bbl.NewID()
	presignedURL, err := app.store.NewUploadURL(r.Context(), objectID, 15*time.Minute)
	if err != nil {
		return err
	}

	res := struct {
		ObjectID string `json:"object_id"`
		URL      string `json:"url"`
	}{
		ObjectID: objectID,
		URL:      presignedURL,
	}
	if err := json.NewEncoder(w).Encode(&res); err != nil {
		return err
	}

	return nil
}
