package app

import (
	"encoding/json"
	"net/http"

	"github.com/ugent-library/bbl"
)

type personSuggestion struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	GivenName  string `json:"given_name,omitempty"`
	FamilyName string `json:"family_name,omitempty"`
}

func (app *App) suggestPeople(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	q := r.URL.Query().Get("q")
	if q == "" {
		return writeJSON(w, []personSuggestion{})
	}

	hits, err := app.services.Index.People().Search(r.Context(), &bbl.SearchOpts{
		Query: q,
		Size:  10,
	})
	if err != nil {
		return err
	}

	suggestions := make([]personSuggestion, 0, len(hits.Hits))
	for _, h := range hits.Hits {
		person, err := app.services.Repo.GetPerson(r.Context(), h.ID)
		if err != nil {
			continue
		}
		suggestions = append(suggestions, personSuggestion{
			ID:         person.ID.String(),
			Name:       h.Name,
			GivenName:  person.GivenName,
			FamilyName: person.FamilyName,
		})
	}

	return writeJSON(w, suggestions)
}

func writeJSON(w http.ResponseWriter, v any) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(v)
}
