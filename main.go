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
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strings"

	_ "bfenty/scanner/docs"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

type Order struct {
	Ordernum         int
	Manufacturer     *string
	ManufacturerName *string
	Status           string
	Comments         *string
	Tracking         *string
	Products         []Product
}

type SortRequest struct {
	ID                int
	SKU               string
	Description       *string
	Instructions      *string
	Weightin          *float64
	Weightout         *float64
	Difference        float64
	DifferencePercent string
	Pieces            *int
	Hours             *float64
	Checkout          *string
	Checkin           *string
	Sorter            string
	Status            string
	ManufacturerPart  *string
	Priority          int
	Warn              bool
}

type Page struct {
	Title         string
	Date          string
	Layout        string
	Message       Message
	Permission    User
	ProductList   []Product
	Orders        []Order
	SortRequests  []SortRequest
	SortRequests2 []SortRequest
	Users         []User
}

type Message struct {
	Success bool
	Title   string
	Body    string
}

type User struct {
	Username    string
	Usercode    int
	Role        string
	Sorting     bool
	Manager     string
	Management  bool
	Permissions Permissions
}

type Permissions struct {
	Admin     bool
	Mgmt      bool
	Receiving bool
	Sorting   bool
}

// initialize Logs
var Logger = logrus.New()

// Handle Messages
func message(r *http.Request) (messagebox Message) {
	if r.URL.Query().Get("messagetitle") != "" {
		messagebox.Body = r.URL.Query().Get("messagebody")
		log.Info("Message: ", messagebox)
	}
	return messagebox
}

func handleerror2(err error, w http.ResponseWriter) Message {
	log.Error(err)
	message := Message{Title: "Error", Body: err.Error(), Success: false}

	// encode message as JSON
	response, _ := json.Marshal(message)

	// set content type to JSON and send response
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)

	// Log the response JSON for debugging
	// log.WithFields(log.Fields{
	// 	"responseJSON": string(response),
	// }).Debug("Error response sent to client")

	return message
}

// format as percent
func formatAsPercent(value float64) string {
	return fmt.Sprintf("%.2f%%", value*100)
}

// Handle Error Messages
func handleerror(err error) (message Message) {
	if err != nil {
		message.Title = "Error"
		message.Success = false
		message.Body = err.Error()
		log.Error(message.Body)
		return message
	}
	message.Success = true
	message.Body = "Success"
	return message
}

// Main function
func main() {
	if os.Getenv("LOGLEVEL") == "DEBUG" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	log.Info("Starting Server")
	var message Message
	db, message = opendb()
	log.Info(message.Body)
	fs := http.FileServer(http.Dir("./docs"))
	http.Handle("/docs/", http.StripPrefix("/docs/", fs))
	http.HandleFunc("/", PageHandler)
	// http.HandleFunc("/login", login)
	// http.HandleFunc("/signup", signup)
	// http.HandleFunc("/logout", Logout)
	// http.HandleFunc("/signin", Signin)
	// http.HandleFunc("/usercreate", Usercreate)
	// http.HandleFunc("/products", Products)
	// http.HandleFunc("/productexist", ProductExist)
	// http.HandleFunc("/productsinsert", ProductInsertion)
	// http.HandleFunc("/productupdate", ProductUpdate)
	// http.HandleFunc("/productdelete", productdelete)
	// http.HandleFunc("/upload", uploadHandler)
	// http.HandleFunc("/export", exportHandler)
	// http.HandleFunc("/reorder", reorder)
	// http.HandleFunc("/reorderapi", ReordersListHandler)
	// http.HandleFunc("/ordercreate", ordercreate)
	// http.HandleFunc("/order", order)
	// http.HandleFunc("/orderlist", orderlist)
	// http.HandleFunc("/orderdelete", orderdelete)
	// http.HandleFunc("/orderupdate", orderupdate)
	// http.HandleFunc("/sorting", Sorting)
	// http.HandleFunc("/checkout", Checkout)
	// http.HandleFunc("/checkin", Checkin)
	// http.HandleFunc("/receiving", Receiving)
	// http.HandleFunc("/sortingupdate", Sortinginsert)
	// http.HandleFunc("/sortrequestdelete", sortrequestdelete)
	// http.HandleFunc("/userupdate", userUpdateHandler)
	// http.HandleFunc("/users", Users)
	// http.HandleFunc("/userdelete", userDeleteHandler)
	// http.HandleFunc("/lookuprequestid", LookupRequestID)
	// http.HandleFunc("/sorterror", SortError)
	// http.HandleFunc("/sorterrorlist", SortErrorList)
	// http.HandleFunc("/sorterrorupdate", sortErrorUpdate)
	// http.HandleFunc("/checkexistingerrors", checkExistingErrors)
	// http.HandleFunc("/update-user", UpdateUser)
	// http.HandleFunc("/dashboard", Dashboard)
	// http.HandleFunc("/reporting", Dashboard)
	// http.HandleFunc("/dashdata", Dashdata)
	// http.HandleFunc("/efficiencydata", Efficiency)

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

	var page Page
	if path != "login" {
		page.Permission = auth(w, r)
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
		Signin(w, r)
	case "products":
		log.Debug("Calling ProductList function")
		ProductList(w, r)
	case "manufacturers":
		log.Debug("Calling ListManufacturers function")
		ListManufacturers(w, r)
	case "productinsert":
		log.Debug("Calling InsertProduct function")
		InsertProduct(w, r)
	case "productdelete":
		log.Debug("Calling DeleteProduct function")
		DeleteProduct(w, r)
	case "productupdate":
		log.Debug("Calling UpdateProduct function")
		UpdateProduct(w, r)
	case "users":
		log.Debug("Calling ListUsersAPI function")
		ListUsersAPI(w, r)
	case "userupdate":
		log.Debug("Calling UserDeleteAPI function")
		UserUpdateAPI(w, r)
	case "customers":
		log.Debug("Calling ListCustomersAPI function")
		ListCustomersAPI(w, r)
	case "sortinglist":
		log.Debug("Calling ListCustomersAPI function")
		ListSortRequestsAPI(w, r)
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
	err := db.QueryRow(query, key).Scan(&exists)
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
	if auth(w, r).Role == "Unauthorized" {
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

// Dashboard
// func Dashboard(w http.ResponseWriter, r *http.Request) {
// 	log.Debug("loading dashboard")
// 	var page Page
// 	page.Permission = auth(w, r)
// 	// page.Message, page.SortRequests = listsortrequests(page.Permission, "receiving", r)
// 	// page.Message, page.Users = listusers("sorting", page.Permission)
// 	page.Message, page.Users = listusers("sorting", page.Permission)
// 	// Get the value of the 'layout' variable from the query string
// 	layout := r.URL.Query().Get("layout")
// 	// Set Page.Layout to the value of the 'layout' variable
// 	page.Layout = layout
// 	t, _ := template.ParseFiles("dashboard.html", "header.html", "login.js")
// 	page.Title = "Dashboard"
// 	t.Execute(w, page)
// }

// func ProductExist(w http.ResponseWriter, r *http.Request) {
// 	r.ParseForm()
// 	log.Info("Checking if product ", r.FormValue("sku"), " exists")
// 	exists, message := ProductExistSQL(r.FormValue("sku"))
// 	fmt.Fprintf(w, exists)
// 	log.Debug(message)
// }

// // Page of list of all orders
// func orderlist(w http.ResponseWriter, r *http.Request) {
// 	var page Page
// 	page.Permission = auth(w, r)
// 	page.Message, page.Orders = listorders(page.Permission)
// 	t, _ := template.ParseFiles("orderlist.html", "header.html", "login.js")
// 	page.Title = "Orders"
// 	t.Execute(w, page)
// }

// // Page of list of all Sort Requests
// func Receiving(w http.ResponseWriter, r *http.Request) {
// 	var page Page
// 	page.Permission = auth(w, r)
// 	page.Message, page.SortRequests = listsortrequests(page.Permission, "receiving", r)
// 	page.Message, page.Users = listusers("sorting", page.Permission)
// 	t, _ := template.ParseFiles("sorting.html", "header.html", "login.js")
// 	page.Title = "Receiving"
// 	t.Execute(w, page)
// }

// // Page of list of all Users
// func Users(w http.ResponseWriter, r *http.Request) {
// 	var page Page
// 	page.Permission = auth(w, r)
// 	page.Message, page.SortRequests = listsortrequests(page.Permission, "receiving", r)
// 	page.Message, page.Users = listusers("all", page.Permission)
// 	t, _ := template.ParseFiles("users2.html", "header.html", "login.js")
// 	page.Title = "User Management"
// 	t.Execute(w, page)
// }

// // Page of list of all Sort Requests
// func Sorting(w http.ResponseWriter, r *http.Request) {
// 	var page Page
// 	page.Permission = auth(w, r)
// 	if page.Permission.Role == "receiving" {
// 		page.Message, page.SortRequests = listsortrequests(page.Permission, "receiving", r)
// 	} else if page.Permission.Role == "admin" {
// 		page.Message, page.SortRequests = listsortrequests(page.Permission, "all", r)
// 	}
// 	page.Message, page.Users = listusers("sorting", page.Permission)
// 	// Get the value of the 'layout' variable from the query string
// 	layout := r.URL.Query().Get("layout")
// 	// Set Page.Layout to the value of the 'layout' variable
// 	page.Layout = layout

// 	t, _ := template.ParseFiles("sorting.html", "header.html", "login.js")
// 	page.Title = "Sorting"
// 	t.Execute(w, page)
// }

// // Page of list of all orders
// func SortError(w http.ResponseWriter, r *http.Request) {
// 	var page Page
// 	page.Permission = auth(w, r)
// 	//page.Message, page.Orders = listorders(page.Permission)
// 	t, _ := template.ParseFiles("sorterror.html", "header.html", "login.js")
// 	page.Title = "Sorting Errors"
// 	t.Execute(w, page)
// }

// // Page to Check Out sorting
// func Checkout(w http.ResponseWriter, r *http.Request) {
// 	var page Page
// 	page.Permission = auth(w, r)
// 	currentTime := time.Now()
// 	page.Date = currentTime.Format("2006-01-02")
// 	page.Message, page.SortRequests = listsortrequests(page.Permission, "checkout", r)
// 	page.Message, page.Users = listusers("sorting", page.Permission)
// 	t, _ := template.ParseFiles("checkout.html", "header.html", "login.js")
// 	page.Title = "Check Out"
// 	t.Execute(w, page)
// }

// // Page to Check Out sorting
// func Checkin(w http.ResponseWriter, r *http.Request) {
// 	var page Page
// 	page.Permission = auth(w, r)
// 	currentTime := time.Now()
// 	page.Date = currentTime.Format("2006-01-02")
// 	page.Message, page.SortRequests2 = listsortrequests(page.Permission, "checkin", r)
// 	t, _ := template.ParseFiles("checkin.html", "header.html", "login.js")
// 	page.Title = "Check In"
// 	t.Execute(w, page)
// }

// // Handle delete order POST request
// func sortrequestdelete(w http.ResponseWriter, r *http.Request) {
// 	var page Page
// 	page.Permission = auth(w, r)
// 	r.ParseForm()
// 	requestid, _ := strconv.Atoi(r.FormValue("requestid"))
// 	sortrequestdeletesql(requestid, page.Permission)
// 	http.Redirect(w, r, r.Header.Get("Referer"), 302)
// }

// // handle POST request for updating order
// func orderupdate(w http.ResponseWriter, r *http.Request) {
// 	var page Page
// 	page.Permission = auth(w, r)
// 	r.ParseForm()
// 	tracking := r.FormValue("tracking")
// 	comment := r.FormValue("comments")
// 	status := r.FormValue("status")
// 	ordernum, _ := strconv.Atoi(r.FormValue("order"))
// 	log.Info("Updating Order ", ordernum, "...")
// 	orderupdatesql(ordernum, tracking, comment, status, page.Permission)
// 	http.Redirect(w, r, r.Header.Get("Referer"), 302)
// }

// // Create a new order
// func ordercreate(w http.ResponseWriter, r *http.Request) {
// 	var page Page
// 	page.Permission = auth(w, r)
// 	log.Debug("Creating Order...")
// 	r.ParseForm()
// 	manufacturer := r.FormValue("manufacturer")

// 	//Create a new order in the system
// 	message, order := nextorder(manufacturer, page.Permission) //create a new order number
// 	for key, values := range r.PostForm {                      //cycle through all the skus and add them to the new order
// 		if key == "sku" {
// 			for _, v := range values {
// 				orderskuadd(order.Ordernum, v, page.Permission)
// 			}
// 		}
// 	}
// 	//redirect to the order view page
// 	http.Redirect(w, r, "/order?order="+strconv.Itoa(order.Ordernum)+"&manufacturer="+manufacturer+"&success="+strconv.FormatBool(message.Success)+"&message="+message.Body, http.StatusSeeOther)
// }

// // Single Order Page
// func order(w http.ResponseWriter, r *http.Request) {
// 	var page Page
// 	page.Permission = auth(w, r)
// 	t, _ := template.ParseFiles("order.html", "header.html", "login.js")
// 	page.Title = "Order"
// 	ordernum, _ := strconv.Atoi(r.URL.Query().Get("order"))
// 	page.Message, page.Orders = orderlookup(ordernum, page.Permission)
// 	if len(page.Orders) == 0 {
// 		http.Redirect(w, r, "/orderlist", http.StatusSeeOther)
// 	}
// 	log.Debug("Order Lookup: ", page.Orders)
// 	t.Execute(w, page)
// }

// // Reorders Page//
// func reorder(w http.ResponseWriter, r *http.Request) {
// 	var page Page
// 	page.Permission = auth(w, r)
// 	t, _ := template.ParseFiles("reorders.html", "header.html", "login.js")
// 	log.Debug("Loading Products...")
// 	page.Title = "Reorders"
// 	page.Message, page.Orders = Reorderlist(page.Permission)
// 	t.Execute(w, page)
// }

// // // Product List Page
// // func Products(w http.ResponseWriter, r *http.Request) {
// // 	var page Page
// // 	page.Permission = auth(w, r)
// // 	t, _ := template.ParseFiles("products.html", "header.html", "login.js")
// // 	log.Debug("Loading Products...")
// // 	page.Title = "Products"
// // 	page.Message, page.ProductList = ProductList(100, r, page.Permission)
// // 	t.Execute(w, page)
// // }

// // Handle export POST request
// func exportHandler(w http.ResponseWriter, r *http.Request) {
// 	var page Page
// 	var filename string
// 	page.Permission = auth(w, r)
// 	r.ParseForm()
// 	ordernum, _ := strconv.Atoi(r.FormValue("order"))
// 	log.Debug("Exporting Excel...")
// 	page.Message, page.Orders = orderlookup(ordernum, page.Permission)
// 	for _, num := range page.Orders {
// 		page.Message, filename = excel(strconv.Itoa(num.Ordernum), num.Products)
// 	}
// 	log.Debug("FILE: ", filename)
// 	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(strconv.Itoa(ordernum)+".xlsx"))
// 	w.Header().Set("Content-Type", "application/octet-stream")
// 	http.ServeFile(w, r, filename)
// 	http.Redirect(w, r, r.Header.Get("Referer"), 302)
// }

// // Handle delete order POST request
// func orderdelete(w http.ResponseWriter, r *http.Request) {
// 	var page Page
// 	page.Permission = auth(w, r)
// 	r.ParseForm()
// 	ordernum, _ := strconv.Atoi(r.FormValue("order"))
// 	orderdeletesql(ordernum, page.Permission)
// 	http.Redirect(w, r, r.Header.Get("Referer"), 302)
// }

// // Handle delete product POST request
// func productdelete(w http.ResponseWriter, r *http.Request) {
// 	var page Page
// 	page.Permission = auth(w, r)
// 	r.ParseForm()
// 	sku := r.FormValue("sku")
// 	productdeletesql(sku, page.Permission)
// 	http.Redirect(w, r, r.Header.Get("Referer"), 302)
// }

// // // Handle update/insert of product POST request
// // func Sortingupdate(w http.ResponseWriter, r *http.Request) {
// // 	var page Page
// // 	page.Permission = auth(w, r)
// // 	log.Debug("Updating sort request...")
// // 	page.Message = Sortinginsert(r, page.Permission)
// // 	http.Redirect(w, r, r.Header.Get("Referer"), 302)
// // }

// // Handle update/insert of product POST request
// func ProductUpdate(w http.ResponseWriter, r *http.Request) {
// 	var page Page
// 	page.Permission = auth(w, r)
// 	log.Debug("Updating Product...")
// 	page.Message = ProductInsert(r, page.Permission)
// 	http.Redirect(w, r, r.Header.Get("Referer"), 302)
// }

// // // New Products Page
// // func ProductInsertion(w http.ResponseWriter, r *http.Request) {
// // 	var page Page
// // 	page.Permission = auth(w, r)
// // 	t, _ := template.ParseFiles("productsinsert.html", "header.html", "login.js")
// // 	log.Debug("Loading Products...")
// // 	page.Title = "New Product"
// // 	page.Message, page.ProductList = ProductList(5, r, page.Permission)
// // 	if r.URL.Query().Get("insert") == "true" {
// // 		page.Message = ProductInsert(r, page.Permission)
// // 	}
// // 	t.Execute(w, page)
// // }

// // Signup Page
// func signup(w http.ResponseWriter, r *http.Request) {
// 	t, _ := template.ParseFiles("signup.html", "header.html", "login.js")
// 	var page Page
// 	page.Title = "Sign Up"
// 	page.Message = message(r)
// 	log.Debug(page)
// 	t.Execute(w, page)
// }
