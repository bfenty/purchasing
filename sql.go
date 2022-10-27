package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

// type Graph struct {
// 	X *string
// 	Y *float64
// 	Z *float64
// }

// type Table struct {
// 	Col1 *string
// 	Col2 *string
// 	Col3 *string
// 	Col4 *string
// 	Col5 *string
// 	Col6 *string
// }

var db *sql.DB

func opendb() (db *sql.DB, messagebox Message) {
	user := os.Getenv("USER")
	pass := os.Getenv("PASS")
	server := os.Getenv("SERVER")
	port := os.Getenv("PORT")
	// Get a database handle.
	var err error
	// var user string
	fmt.Println("Connecting to DB...")
	fmt.Println("user:", user)
	fmt.Println("pass:", pass)
	fmt.Println("server:", server)
	fmt.Println("port:", port)
	fmt.Println("Opening Database...")
	connectstring := os.Getenv("USER") + ":" + os.Getenv("PASS") + "@tcp(" + os.Getenv("SERVER") + ":" + os.Getenv("PORT") + ")/purchasing?parseTime=true"
	fmt.Println("Connection: ", connectstring)
	db, err = sql.Open("mysql",
		connectstring)
	if err != nil {
		messagebox.Success = false
		messagebox.Body = err.Error()
		fmt.Println("Message: ", messagebox.Body)
		return nil, messagebox
	}

	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		return nil, handleerror(pingErr)
	}

	//Success!
	fmt.Println("Returning Open DB...")
	messagebox.Success = true
	messagebox.Body = "Success"
	return db, messagebox
}

// Product List
func ProductList(limit int, r *http.Request) (message Message, products []Product) {
	// Get a database handle.
	var err error

	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, message = opendb()
		return handleerror(pingErr), products
	}

	var sku string
	var i []interface{}
	var newquery string

	sku = r.URL.Query().Get("sku")
	manufacturer := r.URL.Query().Get("manufacturer")
	manufacturerpart := r.URL.Query().Get("manufacturerpart")
	processrequest := r.URL.Query().Get("processrequest")
	sortingrequest := r.URL.Query().Get("sortingrequest")
	unit := r.URL.Query().Get("unit")
	unitprice := r.URL.Query().Get("unitprice")
	currency := r.URL.Query().Get("currency")
	orderqty := r.URL.Query().Get("orderqty")

	//Build the Query
	newquery = "SELECT * FROM `skus` WHERE 1"
	if sku != "" {
		sku += "%"
		i = append(i, sku)
		newquery += " AND sku_internal LIKE ?"
	}
	if manufacturer != "" {
		i = append(i, manufacturer)
		newquery += " AND manufacturer_code = ?"
	}
	if manufacturerpart != "" {
		manufacturerpart += "%"
		i = append(i, manufacturerpart)
		newquery += " AND sku_manufacturer LIKE ?"
	}
	if processrequest != "" {
		processrequest = "%" + processrequest + "%"
		i = append(i, processrequest)
		newquery += " AND processing_request LIKE ?"
	}
	if sortingrequest != "" {
		sortingrequest = "%" + sortingrequest + "%"
		i = append(i, sortingrequest)
		newquery += " AND sorting_request LIKE ?"
	}
	if unit != "" {
		unit = "%" + unit + "%"
		i = append(i, unit)
		newquery += " AND unit LIKE ?"
	}
	if unitprice != "" {
		i = append(i, unitprice)
		newquery += " AND unit_price = ?"
	}
	if currency != "" {
		i = append(i, currency)
		newquery += " AND currency = ?"
	}
	if orderqty != "" {
		i = append(i, orderqty)
		newquery += " AND order_qty = ?"
	}
	newquery += " order by 1 limit ?"

	//Run Query
	i = append(i, limit) //always add the limit to the end
	fmt.Println(i...)    //debug variables map
	fmt.Println("Running Product List")
	fmt.Println(newquery)
	rows, err := db.Query(newquery, i...)
	if err != nil {
		return handleerror(err), products
	}
	defer rows.Close()

	//Pull Data
	for rows.Next() {
		var r Product
		err := rows.Scan(&r.SKU, &r.Manufacturer, &r.ManufacturerPart, &r.Description, &r.ProcessRequest, &r.SortingRequest, &r.Unit, &r.UnitPrice, &r.Currency, &r.Qty)
		if err != nil {
			return handleerror(err), products
		}
		products = append(products, r)
	}

	//Debug Excel
	excel(products)

	return message, products
}

// Product Insert
func ProductInsert(r *http.Request) (message Message) {
	// Get a database handle.
	var err error

	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, message = opendb()
		return handleerror(pingErr)
	}

	var i []interface{}
	var newquery string

	sku := r.URL.Query().Get("sku")
	manufacturer := r.URL.Query().Get("manufacturer")
	manufacturerpart := r.URL.Query().Get("manufacturerpart")
	processrequest := r.URL.Query().Get("processrequest")
	sortingrequest := r.URL.Query().Get("sortingrequest")
	unit := r.URL.Query().Get("unit")
	unitprice := r.URL.Query().Get("unitprice")
	currency := r.URL.Query().Get("currency")
	orderqty := r.URL.Query().Get("orderqty")

	i = append(i, sku)
	i = append(i, manufacturer)
	i = append(i, manufacturerpart)
	i = append(i, processrequest)
	i = append(i, sortingrequest)
	i = append(i, unit)
	i = append(i, unitprice)
	i = append(i, currency)
	i = append(i, orderqty)

	//Build the Query
	newquery = "REPLACE INTO skus (`sku_internal`, `manufacturer_code`, `sku_manufacturer`, `processing_request`, `sorting_request`, `unit`, `unit_price`, `Currency`, `order_qty`) VALUES (?,?,?,?,?,?,?,?,?)"

	//Run Query
	fmt.Println(i...) //debug variables map
	fmt.Println("Running Product List")
	fmt.Println(newquery)
	rows, err := db.Query(newquery, i...)
	if err != nil {
		return handleerror(err)
	}
	defer rows.Close()

	//Pull Data
	for rows.Next() {
		var r Product
		err := rows.Scan(&r.SKU, &r.Manufacturer, &r.ManufacturerPart, &r.Description, &r.ProcessRequest, &r.SortingRequest, &r.Unit, &r.UnitPrice, &r.Currency, &r.Qty)
		if err != nil {
			return handleerror(err)
		}
	}
	message.Title = "Success"
	message.Body = "Successfully inserted row"
	message.Success = true
	return message
}

// Update User Password
func Updatepass(user string, pass string, secret string) (message Message, success bool) {
	pingErr := db.Ping()
	if pingErr != nil {
		return handleerror(pingErr), false
	}

	//Check for secret
	if secret != os.Getenv("SECRET") {
		message.Title = "Secret Auth Failed"
		message.Body = "Secret Auth Failed"
		return message, false
	}

	hashpass := hashAndSalt([]byte(pass))
	fmt.Println("Creating password hash of length ", len(hashpass), ": ", hashpass)
	var newquery string = "update users set password = ? where username = ? and password = ''"
	rows, err := db.Query(newquery, hashpass, user)
	if err != nil {
		return handleerror(err), false
	}
	defer rows.Close()
	message.Title = "Success"
	message.Body = "Success"
	message.Success = true
	return message, true
}

// Authenticate user from DB
func userauth(user string, pass string) (permission string, message Message) {
	// Get a database handle.
	var err error
	var dbpass string
	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		return "notfound", handleerror(pingErr)
	}
	//set Variables
	//Query
	var newquery string = "select password, permissions from users where username = ?"
	// fmt.Println(newquery)
	rows, err := db.Query(newquery, user)
	if err != nil {
		return "notfound", handleerror(err)
	}
	defer rows.Close()
	//Pull Data
	for rows.Next() {
		err := rows.Scan(&dbpass, &permission)
		if err != nil {
			return "notfound", handleerror(err)
		}
	}
	err = rows.Err()
	if err != nil {
		return "notfound", handleerror(err)
	}

	fmt.Println("Checking Permissions: ", permission)
	//If user has not set a password
	if dbpass == "" {
		message.Title = "Set Password"
		message.Body = "Password not set, please create password"
		return "newuser", message
	}

	//If Permissions do not exist for user
	if permission == "" {
		message.Title = "Permission not found"
		message.Body = "Permissions not set for user. Please contact your system administrator."
		return "notfound", message
	}

	if comparePasswords(dbpass, []byte(pass)) {
		message.Title = "Success"
		message.Body = "Successfully logged in"
		message.Success = true
		// permission = "notfound"
		return permission, message
	}
	message.Title = "Login Failed"
	message.Body = "Login Failed"
	permission = "notfound"
	return permission, message
}

// Authenticate user from DB
func userdata(user string) (permission string) {
	// Get a database handle.
	var err error
	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	//set Variables
	//Query
	var newquery string = "select permissions from users where username = ?"
	// fmt.Println(newquery)
	rows, err := db.Query(newquery, user)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	//Pull Data
	for rows.Next() {
		err := rows.Scan(&permission)
		if err != nil {
			log.Fatal(err)
		}
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	if permission == "" {
		return "notfound"
	}

	return permission
}
