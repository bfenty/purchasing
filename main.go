package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
)

type Order struct {
	Ordernum         int
	Manufacturer     *string
	ManufacturerName *string
	Status           *string
	Comments         *string
	Tracking         *string
	Products         []Product
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
	Reorder          *bool
	InventoryQTY     *int
}

type Page struct {
	Title       string
	Message     Message
	Permission  string
	Startdate   string
	Enddate     string
	ProductList []Product
	Orders      []Order
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
	http.HandleFunc("/productupdate", ProductUpdate)
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/export", exportHandler)
	http.HandleFunc("/reorder", reorder)
	http.HandleFunc("/ordercreate", ordercreate)
	http.HandleFunc("/order", order)
	http.HandleFunc("/orderlist", orderlist)
	http.HandleFunc("/orderdelete", orderdelete)
	http.ListenAndServe(":8082", nil)
}

func orderlist(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	page.Message.Body = r.URL.Query().Get("message")
	page.Message.Success, _ = strconv.ParseBool(r.URL.Query().Get("success"))
	page.Message, page.Orders = listorders()
	t, _ := template.ParseFiles("orderlist.html", "header.html", "login.js")
	page.Title = "Orders"
	t.Execute(w, page)
}

func ordercreate(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Creating Order...")
	r.ParseForm()
	manufacturer := r.FormValue("manufacturer")

	//Create a new order in the system
	message, order := nextorder(manufacturer) //create a new order number
	for key, values := range r.PostForm {     //cycle through all the skus and add them to the new order
		if key == "sku" {
			for _, v := range values {
				orderskuadd(order.Ordernum, v)
			}
		}
	}

	//redirect to the order view page
	http.Redirect(w, r, "/order?order="+strconv.Itoa(order.Ordernum)+"&manufacturer="+manufacturer+"&success="+strconv.FormatBool(message.Success)+"&message="+message.Body, http.StatusSeeOther)
}

func order(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	page.Message.Body = r.URL.Query().Get("message")
	page.Message.Success, _ = strconv.ParseBool(r.URL.Query().Get("success"))
	t, _ := template.ParseFiles("order.html", "header.html", "login.js")
	page.Title = "Order"
	ordernum, _ := strconv.Atoi(r.URL.Query().Get("order"))
	page.Message, page.Orders = orderlookup(ordernum)
	if len(page.Orders) == 0 {
		http.Redirect(w, r, "/orderlist", http.StatusSeeOther)
	}
	fmt.Println("Order Lookup: ", page.Orders)
	t.Execute(w, page)
}

func reorder(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	page.Message.Body = r.URL.Query().Get("message")
	if r.URL.Query().Get("success") == "true" {
		page.Message.Success = true
	}
	t, _ := template.ParseFiles("reorders.html", "header.html", "login.js")
	fmt.Println("Loading Products...")
	page.Title = "Reorders"
	page.Message, page.Orders = Reorderlist()
	t.Execute(w, page)
}

func Products(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	page.Message.Body = r.URL.Query().Get("message")
	// fmt.Println("Requested URL: ", strings.SplitN(r.URL.String(), "?", 1)[1])
	if r.URL.Query().Get("success") == "true" {
		page.Message.Success = true
	}
	t, _ := template.ParseFiles("products.html", "header.html", "login.js")
	fmt.Println("Loading Products...")
	page.Title = "Products"
	page.Message, page.ProductList = ProductList(100, r)
	// if r.URL.Query().Get("action") == "export" {
	// 	_, excelproductlist := ProductList(1000, r)
	// 	excel(excelproductlist)
	// }
	t.Execute(w, page)
}

func exportHandler(w http.ResponseWriter, r *http.Request) {
	var page Page
	var filename string
	page.Permission = auth(w, r)
	r.ParseForm()
	ordernum, _ := strconv.Atoi(r.FormValue("order"))
	fmt.Println("Exporting Excel...")
	page.Message, page.Orders = orderlookup(ordernum)
	for _, num := range page.Orders {
		page.Message, filename = excel(strconv.Itoa(num.Ordernum), num.Products)
	}
	fmt.Println("FILE: ", filename)
	// w.Header().Add("Content-Disposition", "Attachment")
	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(strconv.Itoa(ordernum)+".xlsx"))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, filename)
	http.Redirect(w, r, r.Header.Get("Referer"), 302)
}

func orderdelete(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	r.ParseForm()
	ordernum, _ := strconv.Atoi(r.FormValue("order"))
	orderdeletesql(ordernum)
	http.Redirect(w, r, r.Header.Get("Referer"), 302)
}

func ProductUpdate(w http.ResponseWriter, r *http.Request) {
	var page Page
	page.Permission = auth(w, r)
	fmt.Println("Updating Product...")
	page.Message = ProductInsert(r)
	http.Redirect(w, r, r.Header.Get("Referer")+"?message="+page.Message.Body+"&success="+strconv.FormatBool(page.Message.Success), 302)
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
