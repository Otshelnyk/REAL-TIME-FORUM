package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// APIDebugMethod returns effective method for Postman checks.
func APIDebugMethod(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, APIResponse{
		Success: true,
		Data: map[string]string{
			"method": requestMethod(r),
			"path":   r.URL.Path,
		},
	})
}

// APIDebugStatus returns selected HTTP status code in JSON.
func APIDebugStatus(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/api/debug/status/") {
		http.NotFound(w, r)
		return
	}
	codeStr := strings.TrimPrefix(r.URL.Path, "/api/debug/status/")
	code, err := strconv.Atoi(codeStr)
	if err != nil || code < 100 || code > 599 {
		jsonResponse(w, APIResponse{Success: false, Message: "Invalid status code"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(APIResponse{
		Success: code < 400,
		Message: "Debug status response",
		Data: map[string]interface{}{
			"status": code,
			"method": requestMethod(r),
			"path":   r.URL.Path,
		},
	})
}
