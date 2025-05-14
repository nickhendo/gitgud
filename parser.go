package main

import (
	// "gitgud/assert"
	"html/template"
	"log"
	"net/http"
)

var templateFiles = []string{
	"base.html",
	"home.html",
	"repository.html",
}

var funcMap = template.FuncMap{}

var AppTemplates map[string]*template.Template

func ParseTemplates(workDir string) {
	AppTemplates = parseAppTemplates(workDir)
}

func parseAppTemplates(workDir string) map[string]*template.Template {
	templates := make(map[string]*template.Template)
	for _, v := range templateFiles {
		tmpl := template.New(v).Funcs(funcMap)
		template, err := tmpl.ParseFiles(
			workDir+"base.html",
			workDir+v,
		)
		if err != nil {
			log.Panicln(err)
		}
		templates[v] = template
	}
	return templates
}

func RenderNamedAppTemplate(w http.ResponseWriter, r *http.Request, tmpl string, name string, ctx any) error {
	// assert.TemplateFound(tmpl, AppTemplates)
	return RenderNamedTemplate(w, r, AppTemplates, tmpl, name, ctx)
}

func RenderNamedTemplate(w http.ResponseWriter, r *http.Request, tmplBase map[string]*template.Template, tmpl string, name string, ctx any) error {
	return tmplBase[tmpl].ExecuteTemplate(w, name, ctx)
}
