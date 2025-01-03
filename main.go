// @title Purchasing System API
// @description Generic Description
// @version 1.0
// @host 127.0.0.1:8082
// @BasePath /api/
// @schemes http https

// Security Definitions
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key

package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"purchasing/api"
	"purchasing/config"
	"purchasing/models"
	"strings"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

// initialize Logs
var Logger = logrus.New()

// Main function
func main() {
	if os.Getenv("LOGLEVEL") == "DEBUG" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	log.Info("Starting Server")
	var message models.Message
	config.DB, message = config.Opendb()
	log.Info(message.Body)
	fs := http.FileServer(http.Dir("./docs"))
	http.Handle("/docs/", http.StripPrefix("/docs/", fs))
	http.HandleFunc("/", PageHandler)
	http.ListenAndServe(":8082", nil)
}

// Function to check if a file exists
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func PageHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the path component and strip the leading slash
	path := strings.TrimPrefix(r.URL.Path, "/")
	log.WithFields(log.Fields{
		"path": path,
	}).Debug("PageHandler invoked")

	// Delegate to Generic APIHandler for paths starting with 'api-handler'
	if strings.HasPrefix(path, "api-handler") {
		log.Debug("Path starts with 'api-handler', delegating to Generic APIHandler")
		GenericAPIHandler(w, r)
		return
	}

	// Delegate to APIHandler for paths starting with 'api'
	if strings.HasPrefix(path, "api") {
		log.Debug("Path starts with 'api', delegating to APIHandler")
		APIHandler(w, r)
		return
	}

	//if path is blank, redirect to login
	if path == "" {
		path = "login"
	}

	// Construct the template file name
	tmplFile := fmt.Sprintf("html/%s.html", path)

	// Check if the template file exists
	if !fileExists(tmplFile) {
		log.WithFields(log.Fields{
			"file": tmplFile,
		}).Warn("Template file not found")

		// Use http.NotFound to send a 404 response
		http.NotFound(w, r)
		return
	}

	// Construct the template
	t, err := template.ParseFiles(tmplFile, "html/header.html", "js/login.js", "js/scripts.html")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"file":  tmplFile,
		}).Error("Error parsing template files")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var page models.Page
	if path != "login" {
		page.Permission = api.Auth(w, r)
	}
	page.Title = strings.Title(path)          // Capitalize the first letter of the title
	page.Layout = r.URL.Query().Get("layout") // Get the layout from the URL
	log.Debug("Layout:", page.Layout)

	log.WithFields(log.Fields{
		"title":   page.Title,
		"message": page.Message,
	}).Debug("Executing template with page data")

	if err := t.Execute(w, page); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Error executing template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func APIHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("APIHandler called")

	// Extracting and logging the path component
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	log.WithFields(log.Fields{
		"Path": path,
	}).Debug("API path identified")

	// API Key validation
	apiKey := r.Header.Get("X-API-Key") // You can also use r.URL.Query().Get("api_key") if the key is in query params
	if !validateAPIKey(apiKey) && path != "signin" {
		log.WithFields(log.Fields{
			"API Key": apiKey,
		}).Warn("Invalid or missing API Key")
		http.Error(w, "Unauthorized: Invalid API Key", http.StatusUnauthorized)
		return
	}

	// Determining the path and calling the corresponding function
	switch path {
	case "signin":
		log.Debug("Calling Signin function")
		api.Signin(w, r)
	case "products":
		log.Debug("Calling ProductList function")
		api.ProductList(w, r)
	case "manufacturers":
		log.Debug("Calling ListManufacturers function")
		api.ListManufacturers(w, r)
	case "productinsert":
		log.Debug("Calling InsertProduct function")
		api.InsertProduct(w, r)
	case "productdelete":
		log.Debug("Calling DeleteProduct function")
		api.DeleteProduct(w, r)
	case "productupdate":
		log.Debug("Calling UpdateProduct function")
		api.UpdateProduct(w, r)
	case "users":
		log.Debug("Calling ListUsersAPI function")
		api.ListUsersAPI(w, r)
	case "userupdate":
		log.Debug("Calling UserDeleteAPI function")
		api.UserUpdateAPI(w, r)
	case "customers":
		log.Debug("Calling ListCustomersAPI function")
		api.ListCustomersAPI(w, r)
	case "sortinglist":
		log.Debug("Calling ListCustomersAPI function")
		api.ListSortRequestsAPI(w, r)
	case "sorterrorupdate":
		log.Debug("Calling sortErrorUpdate function")
		SortErrorUpdate(w, r)
	// Add other cases as needed
	default:
		log.WithFields(log.Fields{
			"Path": path,
		}).Warn("API path not found, returning 404")
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}

func validateAPIKey(key string) bool {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM purchasing.apikeys WHERE apikey = ?)"

	// Query the database to check if the key exists
	err := config.DB.QueryRow(query, key).Scan(&exists)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to query database for API key")
		return false
	}

	return exists
}

func GenericAPIHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("GenericAPIHandler called")

	// Extract the entire query string
	queryString := r.URL.RawQuery

	// Validate user session or permissions
	if api.Auth(w, r).Role == "Unauthorized" {
		log.Warn("Unauthorized access attempt to GenericAPIHandler")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract target API endpoint
	targetAPI := r.URL.Query().Get("targetAPI")
	if targetAPI == "" {
		log.Warn("Bad Request: Missing targetAPI parameter")
		http.Error(w, "Bad Request: targetAPI parameter is required", http.StatusBadRequest)
		return
	}

	// Construct the full URL if targetAPI is a relative path
	// Assuming you have a base URL for the API
	baseURL := "http://127.0.0.1:8082" // Replace with the actual base URL
	fullURL := baseURL + targetAPI
	if queryString != "" {
		fullURL += "?" + queryString
	}
	log.Debug("API Url being accessed: ", fullURL)

	// Retrieve the API Key for the target API
	// The API key is stored in an environment variable
	apiKey := os.Getenv("APP_API_KEY")
	log.WithFields(log.Fields{
		"API-KEY": apiKey,
	}).Debug("API Key")
	if apiKey == "" {
		log.Error("API Key for the target API is not set")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Forwarding the request
	proxyReq, err := http.NewRequest(r.Method, fullURL, r.Body)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to create new request for proxy")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	// Copy headers from the original request to the proxy request
	copyHeader(r.Header, proxyReq.Header)

	// Set the API key in the request header
	proxyReq.Header.Set("X-API-Key", apiKey)

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to forward request to target API")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	copyHeader(resp.Header, w.Header())
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to copy response body to client")
	}
}

func copyHeader(src, dest http.Header) {
	for key, values := range src {
		for _, value := range values {
			dest.Add(key, value)
		}
	}
}
