package handlers

import (
	"encoding/json"
	"net/http"
)

func LiveMatchUpdate(w http.ResponseWriter, r *http.Request) {
	// Example logic
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Live match updated"})
}
