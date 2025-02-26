package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"purchasing/api"
	"purchasing/config"
	_ "purchasing/docs" // Import Swagger documentation
	"purchasing/models"
	"purchasing/sql"
	"strings"

	"github.com/sirupsen/logrus"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Logger instance
var Logger = logrus.New()

func main() {
	setupLogging()
	logrus.Info("Starting Server")

	var message models.Message
	config.DB, message = config.Opendb()
	logrus.Info(message.Body)

	setupRoutes()

	logrus.Fatal(http.ListenAndServe(":8082", nil))
}

func setupLogging() {
	if os.Getenv("LOGLEVEL") == "DEBUG" {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
}

func setupRoutes() {
	http.Handle("/docs/", http.StripPrefix("/docs/", httpSwagger.Handler(
		httpSwagger.URL("http://127.0.0.1:8082/docs/swagger.json"),
	)))

	http.HandleFunc("/", pageHandler)
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	return err == nil && !info.IsDir()
}

func pageHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	logrus.WithField("path", path).Debug("Handling page request")

	if strings.HasPrefix(path, "api-handler") {
		genericAPIHandler(w, r)
		return
	}

	if strings.HasPrefix(path, "api") {
		apiHandler(w, r)
		return
	}

	if path == "" {
		path = "login"
	}

	tmplFile := fmt.Sprintf("html/%s.html", path)
	if !fileExists(tmplFile) {
		logrus.WithField("file", tmplFile).Warn("Template not found")
		http.NotFound(w, r)
		return
	}

	t, err := template.ParseFiles(tmplFile, "html/header.html", "js/login.js", "js/scripts.html")
	if err != nil {
		logrus.WithError(err).WithField("file", tmplFile).Error("Template parsing error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var page models.Page
	if path != "login" {
		page.Permission = api.Auth(w, r)
	}
	page.Title = strings.Title(path)
	page.Layout = r.URL.Query().Get("layout")

	if err := t.Execute(w, page); err != nil {
		logrus.WithError(err).Error("Template execution error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	logrus.WithField("path", path).Debug("API request received")

	apiKey := r.Header.Get("X-API-Key")
	if !validateAPIKey(apiKey) && path != "signin" {
		logrus.WithField("apiKey", apiKey).Warn("Invalid API Key")
		http.Error(w, "Unauthorized: Invalid API Key", http.StatusUnauthorized)
		return
	}

	apiRoutes := map[string]func(http.ResponseWriter, *http.Request){
		"signin":          api.Signin,
		"logout":          api.Logout,
		"products":        api.ProductList,
		"manufacturers":   api.ListManufacturers,
		"productinsert":   api.InsertProduct,
		"productdelete":   api.DeleteProduct,
		"productupdate":   api.UpdateProduct,
		"users":           api.ListUsersAPI,
		"userupdate":      api.UserUpdateAPI,
		"customers":       api.ListCustomersAPI,
		"sortinglist":     api.ListSortRequestsAPI,
		"sortingupdate":   api.UpdateSortRequestAPI,
		"sorterrorupdate": sql.SortErrorUpdate,
		"export":          api.ExportDataAPI,
	}

	if handler, exists := apiRoutes[path]; exists {
		handler(w, r)
	} else {
		logrus.WithField("path", path).Warn("API endpoint not found")
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}

func validateAPIKey(key string) bool {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM purchasing.apikeys WHERE apikey = ?)"
	err := config.DB.QueryRow(query, key).Scan(&exists)
	if err != nil {
		logrus.WithError(err).Error("Database query error")
		return false
	}
	return exists
}

func genericAPIHandler(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("Handling generic API request")
	targetAPI := r.URL.Query().Get("targetAPI")
	if targetAPI == "" {
		logrus.Warn("Missing targetAPI parameter")
		http.Error(w, "Bad Request: targetAPI parameter required", http.StatusBadRequest)
		return
	}

	baseURL := "http://127.0.0.1:8082"
	fullURL := baseURL + targetAPI
	if r.URL.RawQuery != "" {
		fullURL += "?" + r.URL.RawQuery
	}

	apiKey := os.Getenv("APP_API_KEY")
	if apiKey == "" {
		logrus.Error("API Key not set")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	proxyReq, err := http.NewRequest(r.Method, fullURL, r.Body)
	if err != nil {
		logrus.WithError(err).Error("Request forwarding error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	proxyReq.Header.Set("X-API-Key", apiKey)
	copyHeader(r.Header, proxyReq.Header)

	resp, err := http.DefaultClient.Do(proxyReq)
	if err != nil {
		logrus.WithError(err).Error("API forwarding failed")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	copyHeader(resp.Header, w.Header())
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func copyHeader(src, dest http.Header) {
	for key, values := range src {
		for _, value := range values {
			dest.Add(key, value)
		}
	}
}
