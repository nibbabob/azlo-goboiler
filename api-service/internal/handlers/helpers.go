package handlers

import (
	"azlo-goboiler/internal/config"
	"context"
	"encoding/json"
	"net/http"
)

// --- Helper Functions ---

func getRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(config.RequestIDKey).(string); ok {
		return requestID
	}
	return "unknown"
}
func writeJSON(w http.ResponseWriter, app *config.Application, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		app.Logger.Error().Err(err).Msg("Failed to write JSON response")
	}
}

func writeResponse(w http.ResponseWriter, app *config.Application, status int, success bool, data interface{}, message string) {
	response := map[string]interface{}{
		"success": success,
		"message": message,
	}

	if data != nil {
		response["data"] = data
	}

	if !success {
		response["error"] = message
	}

	writeJSON(w, app, status, response)
}

func writeSuccess(w http.ResponseWriter, app *config.Application, data interface{}, message string) {
	writeResponse(w, app, http.StatusOK, true, data, message)
}

func writeError(w http.ResponseWriter, app *config.Application, status int, message string) {
	writeResponse(w, app, status, false, nil, message)
}
