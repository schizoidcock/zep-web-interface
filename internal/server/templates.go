package server

import (
	"html/template"
	"reflect"
	"time"

	"github.com/Masterminds/sprig/v3"
)

func loadTemplates() (*template.Template, error) {
	// Create base template function map from sprig (like v0.27)
	funcMap := sprig.FuncMap()
	
	// Add custom functions like v0.27
	funcMap["formatTime"] = func(t time.Time) string {
		return t.Format("2006-01-02 15:04:05")
	}
	funcMap["safeLen"] = func(slice interface{}) int {
		if slice == nil {
			return 0
		}
		v := reflect.ValueOf(slice)
		if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
			return v.Len()
		}
		return 0
	}
	
	// Load all templates with functions
	tmpl := template.New("").Funcs(funcMap)

	// Parse template files in correct order (like v0.27)
	patterns := []string{
		"web/templates/layouts/*.html",
		"web/templates/pages/*.html", 
		"web/templates/components/*.html",
		"web/templates/components/layout/*.html",
		"web/templates/components/content/*.html",
	}

	for _, pattern := range patterns {
		if _, err := tmpl.ParseGlob(pattern); err != nil {
			// Skip if pattern doesn't match any files yet
			continue
		}
	}

	return tmpl, nil
}