package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"time"

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
	ID               int
	SKU              string
	Description      *string
	Instructions     *string
	Weightin         *float64
	Weightout        *float64
	Difference       float64
	Pieces           *int
	Hours            *float64
	Checkout         *string
	Checkin          *string
	Sorter           string
	Status           string
	ManufacturerPart *string
	Priority         int
	Warn             bool
}

type Product struct {
	SKU              string
	Description      *string
	Manufacturer     string
	ManufacturerPart *string
	ProcessRequest   *string
	SortingRequest   *string
	Unit             *string
	UnitPrice        *float64
	Currency         string
	Qty              *int
	Modified         *string
	Reorder          bool
	InventoryQTY     *int
	Season           string
	Image            Image
}

type Page struct {
	Title         string
	Date          string
	Message       Message
	Permission    User
	ProductList   []Product
	Orders        []Order
	SortRequests  []SortRequest
	SortRequests2 []SortRequest
	Users         []User
}

// type Permissions struct {
// 	User  string
// 	Perms string
// 	Admin int
// 	Mgmt  int
// }

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
	http.HandleFunc("/", login)
	http.HandleFunc("/login", login)
	http.HandleFunc("/signup", signup)
	http.HandleFunc("/logout", Logout)
	http.HandleFunc("/signin", Signin)
	http.HandleFunc("/usercreate", Usercreate)
	http.HandleFunc("/products", Products)
	http.HandleFunc("/productexist", ProductExist)
	http.HandleFunc("/productsinsert", ProductInsertion)
	http.HandleFunc("/productupdate", ProductUpdate)
	http.HandleFunc("/productdelete", productdelete)
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/export", exportHandler)
	http.HandleFunc("/reorder", reorder)
	http.HandleFunc("/ordercreate", ordercreate)
	http.HandleFunc("/order", order)
	http.HandleFunc("/orderlist", orderlist)
	http.HandleFunc("/orderdelete", orderdelete)
	http.HandleFunc("/orderupdate", orderupdate)
	http.HandleFunc("/sorting", Sorting)
	http.HandleFunc("/checkout", Checkout)
	http.HandleFunc("/checkin", Checkin)
	http.HandleFunc("/receiving", Receiving)
	http.HandleFunc("/sortingupdate", Sortinginsert)
	http.HandleFunc("/sortrequestdelete", sortrequestdelete)
	http.HandleFunc("/userupdate", userUpdateHandler)
	http.HandleFunc("/users", Users)
	http.HandleFunc("/userdelete", userDeleteHandler)
	http.HandleFunc("/lookuprequestid", LookupRequestID)
	http.HandleFunc("/sorterror", SortError)
	http.HandleFunc("/sorterrorupdate", sortErrorUpdate)
	http.HandleFunc("/checkexistingerrors", checkExistingErrors)
	http.HandleFunc("/update-user", UpdateUser)
	http.HandleFunc("/dashbaord", Dashboard)

	http.ListenAndServe(":8082", nil)
}

// Dashboard
func Dashboard(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	// page.Message, page.SortRequests = listsortrequests(page.Permission, "receiving", r)
	// page.Message, page.Users = listusers("sorting", page.Permission)
	t, _ := template.ParseFiles("dashboard.html", "header.html", "login.js")
	page.Title = "Dashboard"
	t.Execute(w, page)
}

func ProductExist(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	log.Info("Checking if product ", r.FormValue("sku"), " exists")
	exists, message := ProductExistSQL(r.FormValue("sku"))
	fmt.Fprintf(w, exists)
	log.Debug(message)
}

// Page of list of all orders
func orderlist(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	page.Message, page.Orders = listorders(page.Permission)
	t, _ := template.ParseFiles("orderlist.html", "header.html", "login.js")
	page.Title = "Orders"
	t.Execute(w, page)
}

// Page of list of all Sort Requests
func Receiving(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	page.Message, page.SortRequests = listsortrequests(page.Permission, "receiving", r)
	page.Message, page.Users = listusers("sorting", page.Permission)
	t, _ := template.ParseFiles("sorting.html", "header.html", "login.js")
	page.Title = "Receiving"
	t.Execute(w, page)
}

// Page of list of all Sort Requests
func Users(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	page.Message, page.SortRequests = listsortrequests(page.Permission, "receiving", r)
	page.Message, page.Users = listusers("all", page.Permission)
	t, _ := template.ParseFiles("users2.html", "header.html", "login.js")
	page.Title = "User Management"
	t.Execute(w, page)
}

// Page of list of all Sort Requests
func Sorting(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	if page.Permission.Role == "receiving" {
		page.Message, page.SortRequests = listsortrequests(page.Permission, "receiving", r)
	} else if page.Permission.Role == "admin" {
		page.Message, page.SortRequests = listsortrequests(page.Permission, "all", r)
	}
	page.Message, page.Users = listusers("sorting", page.Permission)
	t, _ := template.ParseFiles("sorting.html", "header.html", "login.js")
	page.Title = "Sorting"
	t.Execute(w, page)
}

// Page of list of all orders
func SortError(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	//page.Message, page.Orders = listorders(page.Permission)
	t, _ := template.ParseFiles("sorterror.html", "header.html", "login.js")
	page.Title = "Sorting Errors"
	t.Execute(w, page)
}

// Page to Check Out sorting
func Checkout(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	currentTime := time.Now()
	page.Date = currentTime.Format("2006-01-02")
	page.Message, page.SortRequests = listsortrequests(page.Permission, "checkout", r)
	page.Message, page.Users = listusers("sorting", page.Permission)
	t, _ := template.ParseFiles("checkout.html", "header.html", "login.js")
	page.Title = "Check Out"
	t.Execute(w, page)
}

// Page to Check Out sorting
func Checkin(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	currentTime := time.Now()
	page.Date = currentTime.Format("2006-01-02")
	page.Message, page.SortRequests2 = listsortrequests(page.Permission, "checkin", r)
	t, _ := template.ParseFiles("checkin.html", "header.html", "login.js")
	page.Title = "Check In"
	t.Execute(w, page)
}

// Handle delete order POST request
func sortrequestdelete(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	r.ParseForm()
	requestid, _ := strconv.Atoi(r.FormValue("requestid"))
	sortrequestdeletesql(requestid, page.Permission)
	http.Redirect(w, r, r.Header.Get("Referer"), 302)
}

// handle POST request for updating order
func orderupdate(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	r.ParseForm()
	tracking := r.FormValue("tracking")
	comment := r.FormValue("comments")
	status := r.FormValue("status")
	ordernum, _ := strconv.Atoi(r.FormValue("order"))
	log.Info("Updating Order ", ordernum, "...")
	orderupdatesql(ordernum, tracking, comment, status, page.Permission)
	http.Redirect(w, r, r.Header.Get("Referer"), 302)
}

// Create a new order
func ordercreate(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	log.Debug("Creating Order...")
	r.ParseForm()
	manufacturer := r.FormValue("manufacturer")

	//Create a new order in the system
	message, order := nextorder(manufacturer, page.Permission) //create a new order number
	for key, values := range r.PostForm {                      //cycle through all the skus and add them to the new order
		if key == "sku" {
			for _, v := range values {
				orderskuadd(order.Ordernum, v, page.Permission)
			}
		}
	}
	//redirect to the order view page
	http.Redirect(w, r, "/order?order="+strconv.Itoa(order.Ordernum)+"&manufacturer="+manufacturer+"&success="+strconv.FormatBool(message.Success)+"&message="+message.Body, http.StatusSeeOther)
}

// Single Order Page
func order(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	t, _ := template.ParseFiles("order.html", "header.html", "login.js")
	page.Title = "Order"
	ordernum, _ := strconv.Atoi(r.URL.Query().Get("order"))
	page.Message, page.Orders = orderlookup(ordernum, page.Permission)
	if len(page.Orders) == 0 {
		http.Redirect(w, r, "/orderlist", http.StatusSeeOther)
	}
	log.Debug("Order Lookup: ", page.Orders)
	t.Execute(w, page)
}

// Reorders Page
func reorder(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	t, _ := template.ParseFiles("reorders.html", "header.html", "login.js")
	log.Debug("Loading Products...")
	page.Title = "Reorders"
	page.Message, page.Orders = Reorderlist(page.Permission)
	t.Execute(w, page)
}

// Product List Page
func Products(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	t, _ := template.ParseFiles("products.html", "header.html", "login.js")
	log.Debug("Loading Products...")
	page.Title = "Products"
	page.Message, page.ProductList = ProductList(100, r, page.Permission)
	t.Execute(w, page)
}

// Handle export POST request
func exportHandler(w http.ResponseWriter, r *http.Request) {
	var page Page
	var filename string
	page.Permission = auth(w, r)
	r.ParseForm()
	ordernum, _ := strconv.Atoi(r.FormValue("order"))
	log.Debug("Exporting Excel...")
	page.Message, page.Orders = orderlookup(ordernum, page.Permission)
	for _, num := range page.Orders {
		page.Message, filename = excel(strconv.Itoa(num.Ordernum), num.Products)
	}
	log.Debug("FILE: ", filename)
	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(strconv.Itoa(ordernum)+".xlsx"))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, filename)
	http.Redirect(w, r, r.Header.Get("Referer"), 302)
}

// Handle delete order POST request
func orderdelete(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	r.ParseForm()
	ordernum, _ := strconv.Atoi(r.FormValue("order"))
	orderdeletesql(ordernum, page.Permission)
	http.Redirect(w, r, r.Header.Get("Referer"), 302)
}

// Handle delete product POST request
func productdelete(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	r.ParseForm()
	sku := r.FormValue("sku")
	productdeletesql(sku, page.Permission)
	http.Redirect(w, r, r.Header.Get("Referer"), 302)
}

// // Handle update/insert of product POST request
// func Sortingupdate(w http.ResponseWriter, r *http.Request) {
// 	var page Page
// 	page.Permission = auth(w, r)
// 	log.Debug("Updating sort request...")
// 	page.Message = Sortinginsert(r, page.Permission)
// 	http.Redirect(w, r, r.Header.Get("Referer"), 302)
// }

// Handle update/insert of product POST request
func ProductUpdate(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	log.Debug("Updating Product...")
	page.Message = ProductInsert(r, page.Permission)
	http.Redirect(w, r, r.Header.Get("Referer"), 302)
}

// New Products Page
func ProductInsertion(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	t, _ := template.ParseFiles("productsinsert.html", "header.html", "login.js")
	log.Debug("Loading Products...")
	page.Title = "New Product"
	page.Message, page.ProductList = ProductList(5, r, page.Permission)
	if r.URL.Query().Get("insert") == "true" {
		page.Message = ProductInsert(r, page.Permission)
	}
	t.Execute(w, page)
}

// Login page
func login(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("login.html", "header.html", "login.js")
	var page Page
	page.Title = "Login"
	page.Message = message(r)
	log.Debug(page)
	t.Execute(w, page)
}

// Signup Page
func signup(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("signup.html", "header.html", "login.js")
	var page Page
	page.Title = "Sign Up"
	page.Message = message(r)
	log.Debug(page)
	t.Execute(w, page)
}
