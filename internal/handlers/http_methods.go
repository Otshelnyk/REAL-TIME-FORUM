package handlers

import (
	"net/http"
	"strings"
)

// requestMethod returns effective HTTP method and supports method override
// through X-HTTP-Method-Override header or _method form field.
func requestMethod(r *http.Request) string {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method != http.MethodPost {
		return method
	}

	if headerMethod := strings.ToUpper(strings.TrimSpace(r.Header.Get("X-HTTP-Method-Override"))); headerMethod != "" {
		return headerMethod
	}

	_ = r.ParseForm()
	if formMethod := strings.ToUpper(strings.TrimSpace(r.FormValue("_method"))); formMethod != "" {
		return formMethod
	}

	return method
}
