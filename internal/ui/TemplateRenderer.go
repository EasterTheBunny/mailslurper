// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package ui

import (
	"fmt"
	"html/template"
	"io"

	"github.com/labstack/echo/v4"

	"github.com/mailslurper/mailslurper/v2/web"
)

var templates map[string]*template.Template

/*
TemplateRenderer describes a handlers for rendering layouts/pages
*/
type TemplateRenderer struct {
	templates *template.Template
}

/*
NewTemplateRenderer creates a new struct
*/
func NewTemplateRenderer() *TemplateRenderer {
	result := &TemplateRenderer{}
	result.LoadTemplates()

	return result
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data any, _ echo.Context) error {
	var tmpl *template.Template
	var ok bool

	if tmpl, ok = templates[name]; !ok {
		return fmt.Errorf("Cannot find template %s", name)
	}

	return tmpl.ExecuteTemplate(w, "layout", data)
}

func (t *TemplateRenderer) LoadTemplates() {
	templates = make(map[string]*template.Template)

	templates["mainLayout:admin"], _ = template.Must(
		template.New("layout").ParseFS(web.StaticWebAssets, "www/mailslurper/layouts/mainLayout.gohtml"),
	).ParseFS(web.StaticWebAssets, "www/mailslurper/pages/admin.gohtml")

	templates["mainLayout:index"], _ = template.Must(
		template.New("layout").ParseFS(web.StaticWebAssets, "www/mailslurper/layouts/mainLayout.gohtml"),
	).ParseFS(web.StaticWebAssets, "www/mailslurper/pages/index.gohtml")

	templates["mainLayout:manageSavedSearches"], _ = template.Must(
		template.New("layout").ParseFS(web.StaticWebAssets, "www/mailslurper/layouts/mainLayout.gohtml"),
	).ParseFS(web.StaticWebAssets, "www/mailslurper/pages/manageSavedSearches.gohtml")

	templates["loginLayout:login"], _ = template.Must(
		template.New("layout").ParseFS(web.StaticWebAssets, "www/mailslurper/layouts/loginLayout.gohtml"),
	).ParseFS(web.StaticWebAssets, "www/mailslurper/pages/login.gohtml")
}
