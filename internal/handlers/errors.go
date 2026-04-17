package handlers

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func shouldRedirectToSPAError(r *http.Request) bool {
	if r == nil {
		return false
	}
	path := r.URL.Path
	if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/static") || strings.HasPrefix(path, "/ws") {
		return false
	}
	// Only redirect real document navigations (avoid breaking fetch/XHR).
	if strings.ToLower(strings.TrimSpace(r.Header.Get("Sec-Fetch-Dest"))) == "document" {
		return true
	}
	accept := strings.ToLower(r.Header.Get("Accept"))
	return strings.Contains(accept, "text/html")
}

func redirectToSPAError(w http.ResponseWriter, r *http.Request, statusCode int, title, message string) {
	q := url.Values{}
	q.Set("code", strconv.Itoa(statusCode))
	if strings.TrimSpace(title) != "" {
		q.Set("title", title)
	}
	if strings.TrimSpace(message) != "" {
		q.Set("message", message)
	}
	target := "/error?" + q.Encode()
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func localizedError(statusCode int, fallbackTitle, fallbackMessage string) (string, string) {
	switch statusCode {
	case http.StatusBadRequest:
		return "Плохой запрос", "Боги не поняли твой призыв..."
	case http.StatusNotFound:
		return "Страница пала в бою", "Даже воины иногда теряют путь..."
	case http.StatusInternalServerError:
		return "Сервер пал в Рагнарёке", "Кузнецы уже чинят щиты..."
	default:
		return fallbackTitle, fallbackMessage
	}
}

// Render400 renders a 400 Bad Request error
func Render400(w http.ResponseWriter, r *http.Request, message string) {
	title, desc := localizedError(http.StatusBadRequest, "400 Bad Request", message)
	if shouldRedirectToSPAError(r) {
		redirectToSPAError(w, r, http.StatusBadRequest, title, desc)
		return
	}
	http.Error(w, title+": "+desc, http.StatusBadRequest)
}

// Render404 renders a 404 Not Found error
func Render404(w http.ResponseWriter, r *http.Request) {
	title, desc := localizedError(http.StatusNotFound, "404 Not Found", "Requested page was not found.")
	if shouldRedirectToSPAError(r) {
		redirectToSPAError(w, r, http.StatusNotFound, title, desc)
		return
	}
	http.Error(w, title+": "+desc, http.StatusNotFound)
}

// Render405 renders a 405 Method Not Allowed error
func Render405(w http.ResponseWriter, r *http.Request) {
	if shouldRedirectToSPAError(r) {
		redirectToSPAError(w, r, http.StatusMethodNotAllowed, "405 Method Not Allowed", "This HTTP method is not allowed for the requested endpoint.")
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// Render500 renders a 500 Internal Server Error
func Render500(w http.ResponseWriter, r *http.Request, message string) {
	title, desc := localizedError(http.StatusInternalServerError, "500 Internal Server Error", message)
	if shouldRedirectToSPAError(r) {
		redirectToSPAError(w, r, http.StatusInternalServerError, title, desc)
		return
	}
	http.Error(w, title+": "+desc, http.StatusInternalServerError)
}
