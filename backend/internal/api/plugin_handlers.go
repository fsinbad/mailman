package api

import (
	"encoding/json"
	"net/http"
)

// ListPluginsHandler handles the request to list all available plugins.
func (h *APIHandler) ListPluginsHandler(w http.ResponseWriter, r *http.Request) {
	plugins, err := h.pluginManager.ListPlugins()
	if err != nil {
		http.Error(w, "Failed to list plugins", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(plugins); err != nil {
		http.Error(w, "Failed to encode plugins to JSON", http.StatusInternalServerError)
	}
}
