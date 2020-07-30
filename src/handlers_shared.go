package main

import (
	"encoding/base64"
	"errors"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

func serverError(w http.ResponseWriter, err error) {
	message := err.Error()
	log.Println(message)
	if !config.VerboseErrors {
		message = "sorry, a server error has occurred. please contact the admins if this persists"
	}
	http.Error(w, message, http.StatusInternalServerError)
}

func badRequest(w http.ResponseWriter, message string) {
	http.Error(w, message, http.StatusBadRequest)
}

func unauthorised(w http.ResponseWriter) {
	http.Error(w, "unauthorised", http.StatusUnauthorized)
}

func renderView(w http.ResponseWriter, r *http.Request, model interface{}, templateFile, pageTitle string) {
	tmpl := template.New("").Funcs(template.FuncMap{
		"isDevelopment": func() bool { return config.IsDevelopment },
		"currentUser":   func() *user { return currentUser },
		"raw":           func(s string) template.HTML { return template.HTML(s) },
		"formatDate": func(s *string) string {
			if s == nil {
				return ""
			}
			val := *s
			asTime, err := time.Parse(time.RFC3339, val)
			if err != nil {
				return err.Error()
			}
			return asTime.Format("02 Jan 2006 15:04PM")
		}})

	var parsed *template.Template
	var err error

	if config.IsDevelopment {
		parsed, err = tmpl.ParseFiles("templates/_master.html", "templates/"+templateFile)
	} else {
		parsed, err = embeddedPageTemplate(tmpl, templateFile)
	}

	if err != nil {
		serverError(w, err)
	} else if err := parsed.ExecuteTemplate(w, "master", model); err != nil {
		serverError(w, err)
	}
}

func embeddedPageTemplate(baseTemplate *template.Template, templateFilename string) (*template.Template, error) {
	var masterPage, targetPage string
	var exists bool

	if masterPage, exists = embeddedAssets["./templates/_master.html"]; !exists {
		return nil, errors.New("master page template not found")
	}
	if targetPage, exists = embeddedAssets["./templates/"+templateFilename]; !exists {
		return nil, errors.New("target page template not found")
	}

	masterBytes, err := base64.StdEncoding.DecodeString(masterPage)
	if err != nil {
		return nil, err
	}
	pageBytes, err := base64.StdEncoding.DecodeString(targetPage)
	if err != nil {
		return nil, err
	}

	fullContent := append(masterBytes, pageBytes...)
	return baseTemplate.Parse(string(fullContent))
}

func embeddedStaticHandler(w http.ResponseWriter, r *http.Request) {
	file := "." + r.URL.Path

	var fileContent string
	if content, exists := embeddedAssets[file]; exists {
		fileContent = content
	} else {
		http.NotFound(w, r)
		return
	}

	setMimeType(w, r)
	bytes, err := base64.StdEncoding.DecodeString(fileContent)
	if err != nil {
		serverError(w, err)
		return
	}
	w.Write(bytes)
}

func runtimeStaticHandler() http.Handler {
	server := http.FileServer(http.Dir("static"))
	fileHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setMimeType(w, r)
		server.ServeHTTP(w, r)
	})

	return http.StripPrefix("/static/", fileHandler)
}

func setMimeType(w http.ResponseWriter, r *http.Request) {
	headers := w.Header()
	ext := filepath.Ext(r.URL.Path)

	switch ext {
	case ".css":
		headers.Set("Content-Type", "text/css")
	case ".js":
		headers.Set("Content-Type", "application/javascript")
	case ".png":
		headers.Set("Content-Type", "image/png")
	case ".svg":
		headers.Set("Content-Type", "image/svg+xml")
	default:
		return
	}
}
