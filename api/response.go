package api

import (
	"encoding/json"
	"net/http"
)

// APIResponse represents standard API response: { message, result }
type APIResponse struct {
	Message string      `json:"message"`
	Result  interface{} `json:"result"`
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func sendSuccess(w http.ResponseWriter, message string, result interface{}) {
	writeJSON(w, http.StatusOK, APIResponse{
		Message: message,
		Result:  result,
	})
}

func sendError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, APIResponse{
		Message: message,
		Result:  nil,
	})
}

func writeJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}
