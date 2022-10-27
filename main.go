package main

import (
	"fmt"
	"html/template"
	"net/http"
)

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
}

// type OrderDetail struct {
// 	ID       int
// 	Picker   *string
// 	Shipper  *string
// 	Picktime time.Time
// 	Shiptime time.Time
// }

type Page struct {
	Title   string
	Message Message
	// Order       OrderDetail
	Permission string
	Startdate  string
	Enddate    string
	// Graph1      []Graph
	// Graph2      []Graph
	// Graph3      []Graph
	// Graph4      []Graph
	// Graph5      []Graph
	// Graph6      []Graph
	// Table1      []Table
	ProductList []Product
}

type Message struct {
	Success bool
	Title   string
	Body    string
}

func message(r *http.Request) (messagebox Message) {
	if r.URL.Query().Get("messagetitle") != "" {
		messagebox.Body = r.URL.Query().Get("messagebody")
		fmt.Println("Message: ", messagebox)
	}
	return messagebox
}

func handleerror(err error) (message Message) {
	if err != nil {
		message.Title = "Error"
		message.Success = false
		message.Body = err.Error()
		fmt.Println("Message: ", message.Body)
		return message
	}
	message.Success = true
	message.Body = "Success"
	return message
}

func main() {
	fmt.Println("Starting Server...")
	// excel()
	var messagebox Message
	db, messagebox = opendb()
	fmt.Println(messagebox.Body)
	http.HandleFunc("/", login)
	http.HandleFunc("/login", login)
	http.HandleFunc("/signup", signup)
	http.HandleFunc("/logout", Logout)
	http.HandleFunc("/signin", Signin)
	http.HandleFunc("/usercreate", Usercreate)
	http.HandleFunc("/products", Products)
	http.HandleFunc("/productsinsert", ProductInsertion)
	http.HandleFunc("/upload", uploadHandler)
	http.ListenAndServe(":8082", nil)
}

func Products(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	t, _ := template.ParseFiles("products.html", "header.html", "login.js")
	fmt.Println("Loading Products...")
	page.Title = "Products"
	page.Message, page.ProductList = ProductList(100, r)
	t.Execute(w, page)
}

func ProductInsertion(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	t, _ := template.ParseFiles("productsinsert.html", "header.html", "login.js")
	fmt.Println("Loading Products...")
	page.Title = "New Product"
	page.Message, page.ProductList = ProductList(5, r)
	if r.URL.Query().Get("insert") == "true" {
		page.Message = ProductInsert(r)
	}
	t.Execute(w, page)
}

func login(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("login.html", "header.html", "login.js")
	var page Page
	page.Title = "Login"
	page.Message = message(r)
	fmt.Println(page)
	t.Execute(w, page)
}

func signup(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("signup.html", "header.html", "login.js")
	var page Page
	page.Title = "Sign Up"
	page.Message = message(r)
	fmt.Println(page)
	t.Execute(w, page)
}
