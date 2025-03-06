package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"strings"

	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	sessionseal "github.com/lavalleeale/SessionSeal"
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
		"signInvite": func(invite db.OrganizationInvite) string {
			inviteData, err := json.Marshal(invite)
			if err != nil {
				// We have created map so marshalling it should never fail
				panic(err)
			}
			return fmt.Sprintf("%s/acceptInvite?data=%s", os.Getenv("URL"),
				sessionseal.Seal(os.Getenv("JWT_SECRET"), inviteData))
		},
	}

	t = template.Must(template.New("").Funcs(funcMap).ParseFS(templatesFS, "templates/**/*"))
	return t
}
