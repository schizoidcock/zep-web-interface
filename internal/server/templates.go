package server

import (
	"encoding/json"
	"fmt"
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
	// Add missing functions from templates
	funcMap["CommaInt"] = func(i int) string {
		// Simple comma formatting for integers - placeholder
		return fmt.Sprintf("%d", i)
	}
	funcMap["ToJSON"] = func(v interface{}) template.JS {
		// Convert to JSON for template use
		if v == nil {
			return template.JS("{}")
		}
		jsonBytes, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return template.JS("{}")
		}
		return template.JS(string(jsonBytes))
	}
	funcMap["Percent"] = func(part, total float64) float64 {
		// Calculate percentage - placeholder
		if total == 0 {
			return 0
		}
		return (part / total) * 100
	}
	
	// Load all templates with functions
	tmpl := template.New("").Funcs(funcMap)

	// Parse template files in correct order (pages first, then layout)
	patterns := []string{
		"web/templates/pages/*.html",
		"web/templates/components/content/*.html",
		"web/templates/components/*.html",
		"web/templates/components/layout/*.html",
	}

	for _, pattern := range patterns {
		if _, err := tmpl.ParseGlob(pattern); err != nil {
			// Skip if pattern doesn't match any files yet
			continue
		}
	}

	return tmpl, nil
}