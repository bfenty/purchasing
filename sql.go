package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
)

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

func orderdeletesql(order int) (message Message) {
	//Debug
	fmt.Println("Deleting order ", order, "...")

	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, message = opendb()
		return handleerror(pingErr)
	}

	//Build the Query
	newquery := "DELETE FROM `orderskus` WHERE ordernum = ?"
	rows, err := db.Query(newquery, order)
	rows.Close()
	if err != nil {
		return handleerror(err)
	}

	//Build the Query
	newquery = "DELETE FROM `orders` WHERE ordernum = ?"
	rows, err = db.Query(newquery, order)
	rows.Close()
	if err != nil {
		return handleerror(err)
	}

	message.Success = true
	message.Title = "Success"
	message.Body = "Successfully deleted order " + strconv.Itoa(order)
	return message
}

func orderskuadd(order int, sku string) (message Message) {
	//Debug
	fmt.Println("Inserting SKU/Order: ", sku, "/", order)

	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, message = opendb()
		return handleerror(pingErr)
	}
	//Build the Query
	newquery := "REPLACE INTO `orderskus`(`ordernum`, `sku_internal`) VALUES (?,?)"

	rows, err := db.Query(newquery, order, sku)
	rows.Close()
	if err != nil {
		return handleerror(err)
	}

	message.Body = "Successfully inserted SKU " + sku
	message.Success = true
	return message
}

func orderlookup(ordernum int) (message Message, orders []Order) {
	//Debug
	fmt.Println("Getting Order: ", strconv.Itoa(ordernum))

	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, message = opendb()
		return handleerror(pingErr), orders
	}
	//Build the Query
	newquery := "SELECT ordernum,trackingnum,comments,manufacturer FROM `orders` WHERE ordernum = ?"

	orderrows, err := db.Query(newquery, ordernum)
	if err != nil {
		return handleerror(pingErr), orders
	}
	defer orderrows.Close()
	fmt.Println("Orderrows: ", orderrows)
	//Pull Data
	for orderrows.Next() {
		var r Order
		err := orderrows.Scan(&r.Ordernum, &r.Tracking, &r.Comments, &r.Manufacturer)
		if err != nil {
			return handleerror(pingErr), orders
		}
		//Build the Query for the skus in the order
		newquery := "SELECT a.sku_internal,`manufacturer_code`,`sku_manufacturer`,`product_option`,`processing_request`,`sorting_request`,`unit`,`unit_price`,`Currency`,`order_qty`,`modified`,`reorder`,`inventory_qty` FROM orderskus a left join skus b on a.sku_internal = b.sku_internal WHERE a.ordernum = ?"
		skurows, err := db.Query(newquery, r.Ordernum)
		if err != nil {
			return handleerror(pingErr), orders
		}
		fmt.Println("SKUrows: ", skurows)
		var skus []Product
		defer skurows.Close()
		for skurows.Next() {
			var r Product
			err := skurows.Scan(&r.SKU, &r.Manufacturer, &r.ManufacturerPart, &r.Description, &r.ProcessRequest, &r.SortingRequest, &r.Unit, &r.UnitPrice, &r.Currency, &r.Qty, &r.Modified, &r.Reorder, &r.InventoryQTY)
			if err != nil {
				return handleerror(pingErr), orders
			}
			skus = append(skus, r)
		}
		r.Products = skus
		fmt.Println("SKUS: ", skus)
		//Append to the orders
		orders = append(orders, r)
	}

	return message, orders
}

func listorders() (message Message, orders []Order) {
	//Debug
	fmt.Println("Getting Orders...")

	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, message = opendb()
		return handleerror(pingErr), orders
	}
	//Build the Query
	newquery := "SELECT ordernum,trackingnum,comments,manufacturer FROM `orders` WHERE 1"

	//Run Query
	rows, err := db.Query(newquery)
	defer rows.Close()
	if err != nil {
		return handleerror(err), orders
	}

	//Pull Data
	for rows.Next() {
		var r Order
		err := rows.Scan(&r.Ordernum, &r.Tracking, &r.Comments, &r.Manufacturer)
		if err != nil {
			return handleerror(err), orders
		}
		orders = append(orders, r)
	}
	return message, orders
}

func nextorder(manufacturer string) (message Message, order Order) {
	// Get a database handle.
	// var err error
	var ordernum int
	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, message = opendb()
		return handleerror(pingErr), order
	}
	//Build the Query
	newquery := "SELECT MAX(ordernum) ordernum  FROM `orders` WHERE 1"

	rows, err := db.Query(newquery)
	defer rows.Close()
	if err != nil {
		return handleerror(err), order
	}
	// var val string
	if rows.Next() {
		rows.Scan(&ordernum)
	}
	ordernum += 1
	fmt.Println(manufacturer, "-", ordernum)

	//insert new order into database
	newquery = "INSERT INTO orders (`ordernum`,`manufacturer`) VALUES (?,?)"
	orderinsert, err := db.Query(newquery, ordernum, manufacturer)
	orderinsert.Close()
	message.Success = true
	message.Body = "Successfully created order " + manufacturer + "-" + strconv.Itoa(ordernum)
	order.Ordernum = ordernum
	return message, order
}

// Reorders List
func Reorderlist() (message Message, orders []Order) {
	// Get a database handle.
	var err error

	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, message = opendb()
		return handleerror(pingErr), orders
	}
	//Build the Query
	newquery := "SELECT skus.manufacturer_code, manufacturer_code.name FROM `skus` left join manufacturer_code on skus.manufacturer_code = manufacturer_code.code WHERE inventory_qty = 0 and reorder = 1 and manufacturer_code != '' group by skus.manufacturer_code, manufacturer_code.name"

	orderrows, err := db.Query(newquery)
	if err != nil {
		return handleerror(pingErr), orders
	}
	defer orderrows.Close()

	//Pull Data
	for orderrows.Next() {
		var r Order
		err := orderrows.Scan(&r.Manufacturer, &r.ManufacturerName)
		if err != nil {
			return handleerror(pingErr), orders
		}
		//Build the Query for the skus in the order
		newquery := "SELECT `sku_internal`,`manufacturer_code`,`sku_manufacturer`,`product_option`,`processing_request`,`sorting_request`,`unit`,`unit_price`,`Currency`,`order_qty`,`modified`,`reorder`,`inventory_qty` FROM `skus` WHERE inventory_qty = 0 and reorder = 1 and manufacturer_code = ?"
		skurows, err := db.Query(newquery, r.Manufacturer)
		if err != nil {
			return handleerror(pingErr), orders
		}
		var skus []Product
		defer skurows.Close()
		for skurows.Next() {
			var r Product
			err := skurows.Scan(&r.SKU, &r.Manufacturer, &r.ManufacturerPart, &r.Description, &r.ProcessRequest, &r.SortingRequest, &r.Unit, &r.UnitPrice, &r.Currency, &r.Qty, &r.Modified, &r.Reorder, &r.InventoryQTY)
			if err != nil {
				return handleerror(pingErr), orders
			}
			skus = append(skus, r)
		}
		r.Products = skus
		//Append to the orders
		orders = append(orders, r)
	}

	return message, orders

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
	reorder := r.URL.Query().Get("reorder")

	//Build the Query
	newquery = "SELECT `sku_internal`,`manufacturer_code`,`sku_manufacturer`,`product_option`,`processing_request`,`sorting_request`,`unit`,`unit_price`,`Currency`,`order_qty`,`modified`,`reorder`,`inventory_qty` FROM `skus` WHERE 1"
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
	if reorder != "" {
		i = append(i, reorder)
		newquery += " AND reorder = ?"
	}
	newquery += " order by 11 desc, 1 limit ?"

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
		err := rows.Scan(&r.SKU, &r.Manufacturer, &r.ManufacturerPart, &r.Description, &r.ProcessRequest, &r.SortingRequest, &r.Unit, &r.UnitPrice, &r.Currency, &r.Qty, &r.Modified, &r.Reorder, &r.InventoryQTY)
		if err != nil {
			return handleerror(err), products
		}
		products = append(products, r)
	}

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

	//Define Variables
	var i []interface{}
	var newquery string

	sku := r.URL.Query().Get("sku")
	descript := r.URL.Query().Get("description")
	manufacturer := r.URL.Query().Get("manufacturer")
	manufacturerpart := r.URL.Query().Get("manufacturerpart")
	processrequest := r.URL.Query().Get("processrequest")
	sortingrequest := r.URL.Query().Get("sortingrequest")
	unit := r.URL.Query().Get("unit")
	unitprice := r.URL.Query().Get("unitprice")
	currency := r.URL.Query().Get("currency")
	orderqty := r.URL.Query().Get("orderqty")
	reorder := r.URL.Query().Get("reorder")

	//ensure that there are no null numerical values
	if unitprice == "" {
		unitprice = "0"
	}
	if orderqty == "" {
		orderqty = "0"
	}

	//Create the fields to insert
	i = append(i, sku)
	i = append(i, manufacturer)
	i = append(i, manufacturerpart)
	i = append(i, processrequest)
	i = append(i, sortingrequest)
	i = append(i, unit)
	i = append(i, unitprice)
	i = append(i, currency)
	i = append(i, orderqty)
	i = append(i, descript)
	if reorder == "yes" {
		i = append(i, 1)
	} else {
		i = append(i, 0)
	}
	fmt.Println("Reorder: ", reorder)

	//Build the Query
	newquery = "REPLACE INTO skus (`sku_internal`, `manufacturer_code`, `sku_manufacturer`, `processing_request`, `sorting_request`, `unit`, `unit_price`, `Currency`, `order_qty`,`product_option`,`reorder`) VALUES (?,?,?,?,?,?,?,?,?,?,?)"

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
