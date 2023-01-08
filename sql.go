package main

import (
	"database/sql"

	// "log"
	"fmt"
	"net/http"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

var db *sql.DB

func opendb() (db *sql.DB, messagebox Message) {
	var err error
	user := os.Getenv("USER")
	pass := os.Getenv("PASS")
	server := os.Getenv("SERVER")
	port := os.Getenv("PORT")
	// Get a database handle.
	log.Info("Connecting to DB...")
	log.Debug("user:", user)
	log.Debug("pass:", pass)
	log.Debug("server:", server)
	log.Debug("port:", port)
	log.Debug("Opening Database...")
	connectstring := os.Getenv("USER") + ":" + os.Getenv("PASS") + "@tcp(" + os.Getenv("SERVER") + ":" + os.Getenv("PORT") + ")/purchasing?parseTime=true"
	log.Debug("Connection: ", connectstring)
	db, err = sql.Open("mysql",
		connectstring)
	if err != nil {
		messagebox.Success = false
		messagebox.Body = err.Error()
		log.Debug("Message: ", messagebox.Body)
		return nil, messagebox
	}

	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		return nil, handleerror(pingErr)
	}

	//Success!
	log.Info("Returning Open DB...")
	messagebox.Success = true
	messagebox.Body = "Success"
	return db, messagebox
}

func ProductExistSQL(sku string) (exists string, message Message) {
	log.Info("SKU: ", sku)
	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, message = opendb()
		return exists, handleerror(pingErr)
	}

	// sku = "TEST"
	newquery := "SELECT COUNT(*) from skus where sku_internal = ?"
	var count int
	err := db.QueryRow(newquery, sku).Scan(&count)
	if err != nil {
		handleerror(err)
	}
	log.Info("Count: ", count)

	if count == 0 {
		exists = "FALSE"
	} else {
		exists = "TRUE"
	}

	return exists, message
}

func orderdeletesql(order int, permission Permissions) (message Message) {
	//Debug
	log.WithFields(log.Fields{"username": permission.User}).Info("Deleting order ", order, "...")

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

func productdeletesql(sku string, permission Permissions) (message Message) {
	//Debug
	log.WithFields(log.Fields{"username": permission.User}).Info("Deleting SKU ", sku, "...")

	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, message = opendb()
		return handleerror(pingErr)
	}

	//Build the Query
	newquery := "DELETE FROM `skus` WHERE sku_internal = ?"
	rows, err := db.Query(newquery, sku)
	rows.Close()
	if err != nil {
		return handleerror(err)
	}

	message.Success = true
	message.Title = "Success"
	message.Body = "Successfully deleted sku " + sku
	//Logging
	log.WithFields(log.Fields{"username": permission.User}).Info("Deleted Product ", sku)
	return message
}

func orderskuadd(order int, sku string, permission Permissions) (message Message) {
	//Debug
	log.WithFields(log.Fields{"username": permission.User}).Info("Inserting SKU/Order: ", sku, "/", order)

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

func orderlookup(ordernum int, permission Permissions) (message Message, orders []Order) {
	//Debug
	log.WithFields(log.Fields{"username": permission.User}).Debug("Getting Order: ", strconv.Itoa(ordernum))

	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, message = opendb()
		return handleerror(pingErr), orders
	}
	//Build the Query
	newquery := "SELECT ordernum,trackingnum,comments,manufacturer,status FROM `orders` WHERE ordernum = ?"

	orderrows, err := db.Query(newquery, ordernum)
	if err != nil {
		return handleerror(pingErr), orders
	}
	defer orderrows.Close()
	log.WithFields(log.Fields{"username": permission.User}).Debug("Orderrows: ", orderrows)
	//Pull Data
	for orderrows.Next() {
		var r Order
		err := orderrows.Scan(&r.Ordernum, &r.Tracking, &r.Comments, &r.Manufacturer, &r.Status)
		if err != nil {
			return handleerror(pingErr), orders
		}
		//Build the Query for the skus in the order
		newquery := "SELECT a.sku_internal,`manufacturer_code`,`sku_manufacturer`,`product_option`,`processing_request`,`sorting_request`,`unit`,`unit_price`,`Currency`,`order_qty`,`modified`,`reorder`,`inventory_qty`,season FROM orderskus a left join skus b on a.sku_internal = b.sku_internal WHERE a.ordernum = ?"
		skurows, err := db.Query(newquery, r.Ordernum)
		if err != nil {
			return handleerror(pingErr), orders
		}
		log.WithFields(log.Fields{"username": permission.User}).Debug("SKUrows: ", skurows)
		var skus []Product
		defer skurows.Close()
		for skurows.Next() {
			var r Product
			err := skurows.Scan(&r.SKU, &r.Manufacturer, &r.ManufacturerPart, &r.Description, &r.ProcessRequest, &r.SortingRequest, &r.Unit, &r.UnitPrice, &r.Currency, &r.Qty, &r.Modified, &r.Reorder, &r.InventoryQTY, &r.Season)
			if err != nil {
				return handleerror(pingErr), orders
			}
			skus = append(skus, r)
		}
		r.Products = skus
		log.WithFields(log.Fields{"username": permission.User}).Debug("SKUS: ", skus)
		//Append to the orders
		orders = append(orders, r)
	}

	return message, orders
}

func orderupdatesql(order int, tracking string, comment string, status string, permission Permissions) (message Message) {
	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, message = opendb()
		return handleerror(pingErr)
	}

	//Build the Query
	log.WithFields(log.Fields{"username": permission.User}).Debug("Building Query...")
	newquery := "UPDATE `orders` SET `trackingnum`=?,`comments`=?,`status`=? WHERE ordernum = ?"

	//Run Query
	rows, err := db.Query(newquery, tracking, comment, status, order)
	defer rows.Close()
	if err != nil {
		return handleerror(err)
	}
	message.Body = "Successfully updated order " + strconv.Itoa(order)
	message.Success = true
	//Logging
	log.WithFields(log.Fields{"username": permission.User}).Info("Updated Order ", strconv.Itoa(order))
	return message
}

func listusers(role string, permission Permissions) (message Message, users []User) {
	//Debug
	log.WithFields(log.Fields{"username": permission.User}).Debug("Getting users with role ", role, "...")

	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, message = opendb()
		return handleerror(pingErr), users
	}
	//Build the Query
	newquery := "SELECT username from users where permissions='sorting'"

	//Run Query
	rows, err := db.Query(newquery)
	defer rows.Close()
	if err != nil {
		return handleerror(err), users
	}

	//Pull Data
	for rows.Next() {
		var r User
		err := rows.Scan(&r.Username)
		if err != nil {
			return handleerror(err), users
		}
		users = append(users, r)
	}
	return message, users
}

func listorders(permission Permissions) (message Message, orders []Order) {
	//Debug
	log.WithFields(log.Fields{"username": permission.User}).Debug("Getting Orders...")

	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, message = opendb()
		return handleerror(pingErr), orders
	}
	//Build the Query
	newquery := "SELECT ordernum,trackingnum,comments,manufacturer,status FROM `orders` WHERE 1"

	//Run Query
	rows, err := db.Query(newquery)
	defer rows.Close()
	if err != nil {
		return handleerror(err), orders
	}

	//Pull Data
	for rows.Next() {
		var r Order
		err := rows.Scan(&r.Ordernum, &r.Tracking, &r.Comments, &r.Manufacturer, &r.Status)
		if err != nil {
			return handleerror(err), orders
		}
		orders = append(orders, r)
	}
	return message, orders
}

// List of all sorting requests
func listsortrequests(permission Permissions, action string) (message Message, sortrequests []SortRequest) {
	//Debug
	log.WithFields(log.Fields{"username": permission.User}).Debug("Getting Sort Requests...")

	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, message = opendb()
		return handleerror(pingErr), sortrequests
	}

	var newquery string

	//Build the Query
	if action == "all" {
		//retrieves all records
		newquery = "SELECT requestid, sku,description,instructions,weightin,weightout,pieces,hours,checkout,checkint,sorter from sortrequest WHERE 1 order by 1 desc"
	} else if action == "checkout" {
		//retrieves only records that have not been checked out yet
		newquery = "SELECT requestid, sku,description,instructions,weightin,weightout,pieces,hours,checkout,checkint,sorter from sortrequest WHERE 1 and sorter is null or sorter='' order by 1"
	} else if action == "checkin" {
		//retrieves only records that have not been checked out yet
		newquery = "SELECT requestid, sku,description,instructions,weightin,weightout,pieces,hours,checkout,checkint,sorter from sortrequest WHERE 1 and sorter is not null and sorter!='' and (checkint = '' or checkint is null) order by 1"
	} else if action == "receiving" {
		//retrieves only records that have not been checked out yet
		newquery = "SELECT requestid, sku,description,instructions,weightin,weightout,pieces,hours,checkout,checkint,sorter from sortrequest WHERE 1 and sorter is not null and sorter!='' and checkint != '' order by 1 desc"
	}

	//Run Query
	rows, err := db.Query(newquery)
	defer rows.Close()
	if err != nil {
		return handleerror(err), sortrequests
	}

	//Pull Data
	for rows.Next() {
		var r SortRequest
		err := rows.Scan(&r.ID, &r.SKU, &r.Description, &r.Instructions, &r.Weightin, &r.Weightout, &r.Pieces, &r.Hours, &r.Checkout, &r.Checkin, &r.Sorter)
		if err != nil {
			return handleerror(err), sortrequests
		}
		sortrequests = append(sortrequests, r)
	}
	return message, sortrequests
}

// Sorting Insert
func Sortinginsert(r *http.Request, permission Permissions) (message Message) {
	//Test DB Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, message = opendb()
		return handleerror(pingErr)
	}

	//Define Variables
	var i []interface{}
	var newquery string

	//Retrieve variables from POST request
	sku := r.URL.Query().Get("sku")
	id := r.URL.Query().Get("requestid")
	descript := r.URL.Query().Get("description")
	instructions := r.URL.Query().Get("instructions")
	weightin, _ := strconv.Atoi(r.URL.Query().Get("weightin"))
	weightout, _ := strconv.Atoi(r.URL.Query().Get("weightout"))
	pieces, _ := strconv.Atoi(r.URL.Query().Get("pieces"))
	hours := r.URL.Query().Get("hours")
	checkout := r.URL.Query().Get("checkout")
	checkin := r.URL.Query().Get("checkin")
	sorter := r.URL.Query().Get("sorter")

	//ensure that there are no null numerical values
	// if weightin == nil {
	// 	weightin = "0"
	// }
	// if weightout == "" {
	// 	weightout = "0"
	// }
	// if pieces == "" {
	// 	pieces = "0"
	// }
	if hours == "" {
		hours = "0"
	}

	//Create the fields to insert
	i = append(i, sku)
	i = append(i, descript)
	i = append(i, instructions)
	if id != "" {
		i = append(i, id)
	} //ensure that the id isn't null before inserting
	i = append(i, weightin)
	i = append(i, weightout)
	i = append(i, pieces)
	i = append(i, hours)
	i = append(i, checkout)
	i = append(i, checkin)
	i = append(i, sorter)
	log.WithFields(log.Fields{"username": permission.User}).Debug("Inserting Sorting Request: ", i)
	log.WithFields(log.Fields{"username": permission.User}).Debug(i...) //debug variables map
	log.WithFields(log.Fields{"username": permission.User}).Debug("Running Product List")

	//Build the Query
	if id != "" {
		//Run the query if ID isn't null
		newquery = "REPLACE INTO sortrequest (`sku`, `description`, `instructions`, `requestid`,`weightin`, `weightout`, `pieces`, `hours`, `checkout`, `checkint`, `sorter`) VALUES (REPLACE(?,' ',''),?,?,?,?,?,?,?,?,?,?)"
		log.WithFields(log.Fields{"username": permission.User}).Debug("Query: ", newquery)
		rows, err := db.Query(newquery, i...)
		if err != nil {
			return handleerror(err)
		}
		defer rows.Close()
	} else {
		//Run the query to insert a new row
		newquery = "INSERT INTO sortrequest (`sku`, `description`, `instructions`, `weightin`, `weightout`, `pieces`, `hours`, `checkout`, `checkint`, `sorter`) VALUES (REPLACE(?,' ',''),?,?,?,?,?,?,?,?,?)"
		log.WithFields(log.Fields{"username": permission.User}).Debug("Query: ", newquery)
		rows, err := db.Query(newquery, i...)
		if err != nil {
			return handleerror(err)
		}
		defer rows.Close()
	}

	//Logging
	log.WithFields(log.Fields{"username": permission.User}).Info("Inserted Product ", sku)
	message.Title = "Success"
	message.Body = "Successfully inserted row"
	message.Success = true
	return message
}

func nextorder(manufacturer string, permission Permissions) (message Message, order Order) {
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
	log.WithFields(log.Fields{"username": permission.User}).Debug(manufacturer, "-", ordernum)

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
func Reorderlist(permission Permissions) (message Message, orders []Order) {
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
		newquery := "SELECT a.sku_internal,`manufacturer_code`,`sku_manufacturer`,`product_option`,`processing_request`,`sorting_request`,`unit`,`unit_price`,`Currency`,`order_qty`,`modified`,`reorder`,`inventory_qty`,season FROM `skus` a LEFT JOIN (select sku_internal FROM orderskus a left join orders b on a.ordernum = b.ordernum where status != 'Closed') b on a.sku_internal = b.sku_internal WHERE inventory_qty = 0 and reorder = 1 and b.sku_internal is null and manufacturer_code = ?"
		skurows, err := db.Query(newquery, r.Manufacturer)
		if err != nil {
			return handleerror(pingErr), orders
		}
		var skus []Product
		defer skurows.Close()
		for skurows.Next() {
			var r Product
			err := skurows.Scan(&r.SKU, &r.Manufacturer, &r.ManufacturerPart, &r.Description, &r.ProcessRequest, &r.SortingRequest, &r.Unit, &r.UnitPrice, &r.Currency, &r.Qty, &r.Modified, &r.Reorder, &r.InventoryQTY, &r.Season)
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
func ProductList(limit int, r *http.Request, permission Permissions) (message Message, products []Product) {
	// Get a database handle.
	var err error

	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, message = opendb()
		return handleerror(pingErr), products
	}

	queryParams := map[string]string{
		"sku":              r.URL.Query().Get("sku"),
		"manufacturer":     r.URL.Query().Get("manufacturer"),
		"manufacturerpart": r.URL.Query().Get("manufacturerpart"),
		"processrequest":   r.URL.Query().Get("processrequest"),
		"sortingrequest":   r.URL.Query().Get("sortingrequest"),
		"unit":             r.URL.Query().Get("unit"),
		"unitprice":        r.URL.Query().Get("unitprice"),
		"currency":         r.URL.Query().Get("currency"),
		"orderqty":         r.URL.Query().Get("orderqty"),
		"reorder":          r.URL.Query().Get("reorder"),
		"season":           r.URL.Query().Get("season"),
	}

	var i []interface{}
	var newquery string

	newquery = "SELECT `sku_internal`,`manufacturer_code`,`sku_manufacturer`,`product_option`,`processing_request`,`sorting_request`,`unit`,`unit_price`,`Currency`,`order_qty`,`modified`,`reorder`,`inventory_qty`,season,url_standard,url_thumb,url_tiny FROM `skus` WHERE 1"

	for param, value := range queryParams {
		if value != "" {
			if param == "sku" || param == "manufacturerpart" || param == "unit" {
				value += "%"
			} else if param == "processrequest" || param == "sortingrequest" {
				value = "%" + value + "%"
			}
			i = append(i, value)
			newquery += fmt.Sprintf(" AND %s = ?", param)
		}
	}

	newquery += " order by 11 desc, 1 limit ?"
	i = append(i, limit)
	log.WithFields(log.Fields{"username": permission.User}).Debug(i...) //debug variables map
	log.WithFields(log.Fields{"username": permission.User}).Debug("Running Product List")
	log.WithFields(log.Fields{"username": permission.User}).Debug(newquery)
	rows, err := db.Query(newquery, i...)
	if err != nil {
		return handleerror(err), products
	}
	defer rows.Close()

	//Pull Data
	for rows.Next() {
		var r Product
		err := rows.Scan(&r.SKU, &r.Manufacturer, &r.ManufacturerPart, &r.Description, &r.ProcessRequest, &r.SortingRequest, &r.Unit, &r.UnitPrice, &r.Currency, &r.Qty, &r.Modified, &r.Reorder, &r.InventoryQTY, &r.Season, &r.Image.URL_Standard, &r.Image.URL_Thumb, &r.Image.URL_Tiny)
		if err != nil {
			return handleerror(err), products
		}
		products = append(products, r)
	}

	return message, products
}

func sortrequestdeletesql(requestid int, permission Permissions) (message Message) {
	//Debug
	log.WithFields(log.Fields{"username": permission.User}).Info("Deleting order ", order, "...")

	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, message = opendb()
		return handleerror(pingErr)
	}

	//Build the Query
	newquery := "DELETE FROM `sortrequest` WHERE requestid = ?"
	rows, err := db.Query(newquery, requestid)
	rows.Close()
	if err != nil {
		return handleerror(err)
	}

	message.Success = true
	message.Title = "Success"
	message.Body = "Successfully deleted Sorting Request ID  " + strconv.Itoa(requestid)
	return message
}

// Product Insert
func ProductInsert(r *http.Request, permission Permissions) (message Message) {
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
	season := r.URL.Query().Get("season")

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
	i = append(i, season)
	log.WithFields(log.Fields{"username": permission.User}).Debug("Reorder: ", reorder)

	//Build the Query
	newquery = "REPLACE INTO skus (`sku_internal`, `manufacturer_code`, `sku_manufacturer`, `processing_request`, `sorting_request`, `unit`, `unit_price`, `Currency`, `order_qty`,`product_option`,`reorder`,season) VALUES (REPLACE(?,' ',''),?,?,?,?,?,?,?,?,?,?,?)"

	//Run Query
	log.WithFields(log.Fields{"username": permission.User}).Debug(i...) //debug variables map
	log.WithFields(log.Fields{"username": permission.User}).Debug("Running Product List")
	log.WithFields(log.Fields{"username": permission.User}).Debug(newquery)
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

	//add image and qty to new row
	qty(sku)

	//Logging
	log.WithFields(log.Fields{"username": permission.User}).Info("Inserted Product ", sku)
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
	log.Debug("Creating password hash of length ", len(hashpass), ": ", hashpass)
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
	// log.WithFields(log.Fields{"username": permission.User}).Debug(newquery)
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

	log.Debug("Checking Permissions: ", permission)
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
func userdata(user string) (permission Permissions) {
	permission.User = user
	// Get a database handle.
	var err error
	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		handleerror(pingErr)
	}
	//set Variables
	//Query
	var newquery string = "select permissions from users where username = ?"
	// log.WithFields(log.Fields{"username": permission.User}).Debug(newquery)
	rows, err := db.Query(newquery, user)
	if err != nil {
		handleerror(err)
	}
	defer rows.Close()
	//Pull Data
	for rows.Next() {
		err := rows.Scan(&permission.Perms)
		if err != nil {
			handleerror(err)
		}
	}
	err = rows.Err()
	if err != nil {
		handleerror(err)
	}
	if permission.Perms == "" {
		permission.Perms = "notfound"
		return permission
	}

	return permission
}

// Update QTY and IMG for products
func QTYUpdate(skus []sku) {

	for i := range skus {
		var newquery string = "UPDATE `skus` SET `inventory_qty`=?,url_thumb=?,url_standard=?,url_tiny=? WHERE sku_internal=REPLACE(?,' ','')"
		rows, err := db.Query(newquery, skus[i].Qty, skus[i].Skuimage.URL_Thumb, skus[i].Skuimage.URL_Standard, skus[i].Skuimage.URL_Tiny, skus[i].SKU)
		defer rows.Close()
		if err != nil {
			log.Error("Message: ", err.Error())
			rows.Close()
		}
		err = rows.Err()
		if err != nil {
			log.Error("Message: ", err.Error())
			rows.Close()
		}
		rows.Close()
	}
}
