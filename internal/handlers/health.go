package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
)

type HealthResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := HealthResponse{
		Message: "Chat service is running",
		Status:  "ok",
	}

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Fatal(err)
	}
}

func DatabaseHealthCheck(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			response := HealthResponse{
				Message: "Database is unavailable",
				Status:  "error",
			}

			if err := json.NewEncoder(w).Encode(response); err != nil {
				log.Fatal(err)
			}
		}

		response := HealthResponse{
			Message: "Database is available",
			Status:  "ok",
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Fatal(err)
		}
	}
}
