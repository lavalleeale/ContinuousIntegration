package main

import (
	"bytes"
	"html/template"
	"log"
	"strings"

	"github.com/lavalleeale/ContinuousIntegration/lib/db"
)

func GetTemplate() *template.Template {
	var t *template.Template

	funcMap := template.FuncMap{
		"Deref": func(i *int) int { return *i },
		"Include": func(name string, data any) (template.HTML, error) {
			buf := &bytes.Buffer{}
			err := t.ExecuteTemplate(buf, name, data)
			return template.HTML(strings.ReplaceAll(
				strings.ReplaceAll(template.HTMLEscapeString(buf.String()), "&#34;", "'"), "\n", "")), err
		},
		"Escape": func(s string) template.HTML {
			t := strings.ReplaceAll(template.HTMLEscapeString(s), "&#34;", "'")
			log.Println(t)
			return template.HTML(t)
		},
		"Arr": func(els ...any) []any {
			return els
		},
		"reverse": func(items []db.Build) []db.Build {
			for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
				items[i], items[j] = items[j], items[i]
			}
			return items
		},
	}

	t = template.Must(template.New("").Funcs(funcMap).ParseFS(templatesFS, "templates/**/*"))
	return t
}
