package handlers

import (
	"fmt"
	"html/template"
)

// InitTemplates initializes the template system
func InitTemplates() {
	Templates = template.Must(template.New("").Funcs(template.FuncMap{
		"iterate": func(count int) []int {
			s := make([]int, count)
			for i := 0; i < count; i++ {
				s[i] = i
			}
			return s
		},
		"add": func(a, b int) int { return a + b },
		"dec": func(a int) int {
			if a > 1 {
				return a - 1
			}
			return 1
		},
		"inc": func(a int) int { return a + 1 },
		"paginationURL": func(page int, category, filter string) string {
			params := ""
			if category != "" {
				params += fmt.Sprintf("category=%v&", category)
			}
			if filter != "" {
				params += fmt.Sprintf("filter=%v&", filter)
			}
			if page > 1 {
				params += fmt.Sprintf("page=%d", page)
			} else {
				if len(params) > 0 && params[len(params)-1] == '&' {
					params = params[:len(params)-1]
				}
			}
			if params == "" {
				return "/"
			}
			if params[len(params)-1] == '&' {
				params = params[:len(params)-1]
			}
			return "/?" + params
		},
	}).ParseGlob("web/templates/*.html"))
}
