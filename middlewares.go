package gokit

import (
	"encoding/json"
	"net/http"
)

type APIFunc func(w http.ResponseWriter, r *http.Request) error

func Public(h APIFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			err := MapError(err)

			w.WriteHeader(err.StatusCode)
			if err := json.NewEncoder(w).Encode(err); err != nil {
				http.Error(w, "Error encoding response", http.StatusInternalServerError)
			}
		}
	}
}
