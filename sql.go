package main

import (
	"database/sql"
	"encoding/json"
	"math"

	// "log"
	"fmt"
	"net/http"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

type Request struct {
	Sorter      string `json:"sorter"`
	Description string `json:"description"`
}

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

func LookupRequestID(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("requestid")

	// Query the database for the sorter and description based on the requestid
	var sorter, description string
	err := db.QueryRow("SELECT sorter, description FROM purchasing.sortrequest WHERE requestid = ?", requestID).Scan(&sorter, &description)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the sorter and description as a JSON object
	jsonObj := map[string]string{"sorter": sorter, "description": description}
	jsonBytes, err := json.Marshal(jsonObj)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonBytes)
}

func sortErrorUpdate(w http.ResponseWriter, r *http.Request) {
	fmt.Println("TEST")
	log.Debug("Inserting Error")
	// Parse the form data
	err := r.ParseForm()
	if err != nil {
		log.WithError(err).Error("Error parsing form data")
		http.Error(w, "Error parsing form data", http.StatusBadRequest)
		return
	}

	// Get the form data
	requestid := r.Form.Get("requestid")
	sorter := r.Form.Get("sorter")
	description := r.Form.Get("description")
	errortype := r.Form.Get("errortype")
	notes := r.Form.Get("notes")

	// Insert the error into the database
	result, err := db.Exec("REPLACE INTO purchasing.sorterror (requestid, errortype, notes) VALUES (?, ?, ?)", requestid, errortype, notes)
	if err != nil {
		log.WithError(err).Error("Error inserting error into database")
		http.Error(w, "Error inserting error into database", http.StatusInternalServerError)
		return
	}

	// Get the ID of the inserted error
	errorID, err := result.LastInsertId()
	if err != nil {
		log.WithError(err).Error("Error getting last inserted ID")
		http.Error(w, "Error getting last inserted ID", http.StatusInternalServerError)
		return
	}

	// Create a JSON response
	response := map[string]interface{}{
		"errorid":     errorID,
		"requestid":   requestid,
		"sorter":      sorter,
		"description": description,
		"errortype":   errortype,
		"notes":       notes,
	}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.WithError(err).Error("Error marshaling JSON response")
		http.Error(w, "Error marshaling JSON response", http.StatusInternalServerError)
		return
	}

	// Set the content type header and write the response
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)

	log.WithFields(log.Fields{
		"errorid":     errorID,
		"requestid":   requestid,
		"sorter":      sorter,
		"description": description,
		"errortype":   errortype,
		"notes":       notes,
	}).Debug("Error reported successfully")
}

func checkExistingErrors(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("requestid")

	// Query the database for any errors with the given request ID
	rows, err := db.Query("SELECT errorid, errortype, notes FROM sorterror WHERE requestid = ?", requestID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Iterate through the rows and build a slice of error objects
	var errors []map[string]interface{}
	for rows.Next() {
		var errorID int
		var errortype, notes string
		err = rows.Scan(&errorID, &errortype, &notes)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Build a map object for the error and append it to the errors slice
		errorObj := map[string]interface{}{
			"errorid":   errorID,
			"errortype": errortype,
			"notes":     notes,
		}
		errors = append(errors, errorObj)
	}
	err = rows.Err()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Encode the errors slice as a JSON object and return it in the response
	jsonBytes, err := json.Marshal(errors)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonBytes)
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

func orderdeletesql(order int, user User) (message Message) {
	//Debug
	log.WithFields(log.Fields{"username": user.Username}).Info("Deleting order ", order, "...")

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

func productdeletesql(sku string, user User) (message Message) {
	//Debug
	log.WithFields(log.Fields{"username": user.Username}).Info("Deleting SKU ", sku, "...")

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
	log.WithFields(log.Fields{"username": user.Username}).Info("Deleted Product ", sku)
	return message
}

func orderskuadd(order int, sku string, user User) (message Message) {
	//Debug
	log.WithFields(log.Fields{"username": user.Username}).Info("Inserting SKU/Order: ", sku, "/", order)

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

func orderlookup(ordernum int, user User) (message Message, orders []Order) {
	//Debug
	log.WithFields(log.Fields{"username": user.Username}).Debug("Getting Order: ", strconv.Itoa(ordernum))

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
	log.WithFields(log.Fields{"username": user.Username}).Debug("Orderrows: ", orderrows)
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
		log.WithFields(log.Fields{"username": user.Username}).Debug("SKUrows: ", skurows)
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
		log.WithFields(log.Fields{"username": user.Username}).Debug("SKUS: ", skus)
		//Append to the orders
		orders = append(orders, r)
	}

	return message, orders
}

func orderupdatesql(order int, tracking string, comment string, status string, user User) (message Message) {
	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, message = opendb()
		return handleerror(pingErr)
	}

	//Build the Query
	log.WithFields(log.Fields{"username": user.Username}).Debug("Building Query...")
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
	log.WithFields(log.Fields{"username": user.Username}).Info("Updated Order ", strconv.Itoa(order))
	return message
}

func listusers(role string, user User) (message Message, users []User) {
	//Debug
	log.WithFields(log.Fields{"username": user.Username}).Debug("Getting users with role ", role, "...")

	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, message = opendb()
		log.Debug(pingErr)
		return handleerror(pingErr), users
	}

	var newquery string
	//Build the Query
	if role == "sorting" {
		newquery = "SELECT username,usercode,permissions,sorting,manager,management from orders.users where sorting=1 and active=1"
	} else if role == "manager" {
		newquery = "SELECT username,usercode,permissions,sorting,manager,management from orders.users where management=1 and active=1"
	} else {
		newquery = "SELECT username,usercode,permissions,sorting,manager,management from orders.users where active=1"
	}

	//Run Query
	rows, err := db.Query(newquery)
	defer rows.Close()
	if err != nil {
		log.Debug(err)
		return handleerror(err), users
	}

	//Pull Data
	for rows.Next() {
		var r User
		err := rows.Scan(&r.Username, &r.Usercode, &r.Role, &r.Sorting, &r.Manager, &r.Management)
		if err != nil {
			log.Debug(err)
			return handleerror(err), users
		}
		users = append(users, r)
	}
	return message, users
}

func UpdateUser(w http.ResponseWriter, r *http.Request) {

	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, _ = opendb()
		// return handleerror(pingErr)
	}

	// Get the form values from the request
	username := r.FormValue("username")
	usercode := r.FormValue("usercode")
	role := r.FormValue("role")
	manager := r.FormValue("manager")
	log.Debug("username:", username, " usercode:", usercode, " role:", role, " manager:", manager)
	var sorting int
	if r.FormValue("sorting") == "true" {
		sorting = 1
	} else {
		sorting = 0
	}
	// sorting, _ := strconv.ParseBool(r.FormValue("sorting"))
	println(username, usercode, role, sorting)

	// If usercode is empty, find the current max value and increment it
	if usercode == "" {
		var maxUsercode int
		err := db.QueryRow("SELECT MAX(usercode) FROM orders.users").Scan(&maxUsercode)
		if err != nil {
			// Handle error
			handleerror(err)
		}
		usercode = strconv.Itoa(maxUsercode + 1)
	}

	// Prepare the SQL statement for inserting the data
	//Logging
	log.Info("Creating Query")
	newquery := "REPLACE INTO orders.users (username, usercode, permissions, sorting,manager) VALUES (?, ?, ?, ?, ?)"

	// Execute the SQL statement with the form values
	log.Info("Executing Query")
	rows, err := db.Query(newquery, username, usercode, role, sorting, manager)
	defer rows.Close()

	if err != nil {
		// Handle error
		println(err)
		http.Error(w, "Failed to update user information.", http.StatusInternalServerError)
		return
	}

	// Redirect the user to the users page
	// http.Redirect(w, r, "/users", http.StatusSeeOther)

	// Update user information in database
	// _, err = db.Exec("UPDATE orders.users SET username=?, permissions=?, manager=?, sorting=? WHERE usercode=?", username, role, manager, sorting, usercode)
	// if err != nil {
	// 	http.Error(w, "Failed to update user information.", http.StatusInternalServerError)
	// 	return
	// }

	// Return success message to client
	w.Write([]byte("User information updated successfully."))
}

func userUpdateHandler(w http.ResponseWriter, r *http.Request) {

	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, _ = opendb()
		// return handleerror(pingErr)
	}

	// Get the form values from the request
	username := r.FormValue("username")
	usercode := r.FormValue("usercode")
	role := r.FormValue("role")
	manager := r.FormValue("manager")
	var sorting int
	if r.FormValue("sorting") == "true" {
		sorting = 1
	} else {
		sorting = 0
	}
	// sorting, _ := strconv.ParseBool(r.FormValue("sorting"))
	println(username, usercode, role, sorting)

	// If usercode is empty, find the current max value and increment it
	if usercode == "" {
		var maxUsercode int
		err := db.QueryRow("SELECT MAX(usercode) FROM orders.users").Scan(&maxUsercode)
		if err != nil {
			// Handle error
			handleerror(err)
		}
		usercode = strconv.Itoa(maxUsercode + 1)
	}

	// Prepare the SQL statement for inserting the data
	//Logging
	log.Info("Creating Query")
	newquery := "REPLACE INTO orders.users (username, usercode, permissions, sorting,manager) VALUES (?, ?, ?, ?, ?)"

	// Execute the SQL statement with the form values
	log.Info("Executing Query")
	rows, err := db.Query(newquery, username, usercode, role, sorting, manager)
	defer rows.Close()

	if err != nil {
		// Handle error
		println(err)
	}

	// Redirect the user to the users page
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func userDeleteHandler(w http.ResponseWriter, r *http.Request) {
	// Get the usercode value from the form
	usercode := r.FormValue("usercode")

	// Prepare the SQL statement for deleting the user
	stmt, err := db.Prepare("UPDATE orders.users SET active = 0 WHERE usercode = ?")
	if err != nil {
		// Handle error
		println(err)
		http.Error(w, "Failed to update user information.", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	// Execute the SQL statement with the usercode value
	_, err = stmt.Exec(usercode)
	if err != nil {
		// Handle error
		println(err)
		http.Error(w, "Failed to update user information.", http.StatusInternalServerError)
		return
	}

	// Redirect the user to the users page
	w.Write([]byte("User information updated successfully."))
}

func listorders(user User) (message Message, orders []Order) {
	//Debug
	log.WithFields(log.Fields{"username": user.Username}).Debug("Getting Orders...")

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
func listsortrequests(user User, action string, r *http.Request) (message Message, sortrequests []SortRequest) {
	//Debug
	log.WithFields(log.Fields{"username": user.Username}).Debug("Getting Sort Requests...")

	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, message = opendb()
		return handleerror(pingErr), sortrequests
	}

	//Gather Search Parameters
	queryParams := map[string]string{
		"sku":              r.URL.Query().Get("sku"),
		"description":      r.URL.Query().Get("description"),
		"sku_manufacturer": r.URL.Query().Get("manufacturerpart"),
		"instructions":     r.URL.Query().Get("instructions"),
		"weightout":        r.URL.Query().Get("weightout"),
		"weightin":         r.URL.Query().Get("weightin"),
		"pieces":           r.URL.Query().Get("pieces"),
		"hours":            r.URL.Query().Get("hours"),
		"checkout":         r.URL.Query().Get("checkout"),
		"checkint":         r.URL.Query().Get("checkin"),
		"sorter":           r.URL.Query().Get("sorter"),
		"status":           r.URL.Query().Get("status"),
		"priority":         r.URL.Query().Get("priority"),
	}

	var i []interface{}
	var newquery string

	//Build the Query
	if action == "all" {
		//retrieves all records
		newquery = "SELECT requestid, sku,description,instructions,weightin,weightout,pieces,hours,checkout,checkint,COALESCE(sorter,''),status,sku_manufacturer,prty from sortrequest WHERE 1 "
		for param, value := range queryParams {
			if value != "" {
				i = append(i, value)
				newquery += fmt.Sprintf(" AND %s = ?", param)
			}
		}
		newquery += " order by 1 desc"
	} else if action == "checkout" {
		//retrieves only records that have not been checked out yet
		newquery = "SELECT requestid, sku,description,instructions,weightin,weightout,pieces,hours,checkout,checkint,COALESCE(sorter,''),status,sku_manufacturer,prty from sortrequest WHERE status = 'new'"
		// if permission.Perms == "sorting" {
		// 	newquery += " AND sorter = '" + user.Username + "'"
		// }
		newquery += " order by prty desc, 1"
	} else if action == "checkin" {
		//retrieves only records that have not been checked in yet
		newquery = "SELECT requestid, sku,description,instructions,weightin,weightout,pieces,hours,checkout,checkint,COALESCE(sorter,''),status,sku_manufacturer,prty from sortrequest WHERE status = 'checkout'"
		if user.Role == "sorting" {
			newquery += " AND sorter = '" + user.Username + "'"
		}
		newquery += " order by 1"
	} else if action == "receiving" {
		//retrieves only records that have been checked back in
		newquery = "SELECT requestid, sku,description,instructions,weightin,weightout,pieces,hours,checkout,checkint,COALESCE(sorter,''),status,sku_manufacturer,prty from sortrequest WHERE status = 'checkin' order by 1 desc"
	}

	newquery += " limit 100"

	//Run Query
	log.WithFields(log.Fields{"username": user.Username}).Debug(i...) //debug variables map
	log.WithFields(log.Fields{"username": user.Username}).Debug("Running Product List")
	log.WithFields(log.Fields{"username": user.Username}).Debug(newquery)
	log.WithFields(log.Fields{"username": user.Username}).Debug(user.Role)
	rows, err := db.Query(newquery, i...)
	defer rows.Close()
	if err != nil {
		return handleerror(err), sortrequests
	}

	//Pull Data
	for rows.Next() {
		var r SortRequest
		err := rows.Scan(&r.ID, &r.SKU, &r.Description, &r.Instructions, &r.Weightin, &r.Weightout, &r.Pieces, &r.Hours, &r.Checkout, &r.Checkin, &r.Sorter, &r.Status, &r.ManufacturerPart, &r.Priority)
		if err != nil {
			return handleerror(err), sortrequests
		}
		var a float64
		var b float64
		var c float64
		a = *r.Weightin
		b = *r.Weightout
		if r.Pieces != nil {
			c = float64(*r.Pieces)
		} else {
			// Handle the case where r.Pieces is nil
			c = 0.0
			fmt.Println("r.Pieces is nil")
		}
		r.Difference = a - b - (c * 0.4555)
		r.Difference = math.Round(r.Difference*100) / 100 // Round to 2 decimal places
		if r.Difference < (-0.1*a) && a != 0 {
			r.Warn = true
		}
		log.Info(c)
		sortrequests = append(sortrequests, r)
	}
	return message, sortrequests
}

// Sorting Insert
func Sortinginsert(r *http.Request, user User) (message Message) {
	//Test DB Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, message = opendb()
		return handleerror(pingErr)
	}

	// //Define Variables
	// var i []interface{}

	// //Retrieve variables from POST request
	// sku := r.URL.Query().Get("sku")
	// id := r.URL.Query().Get("requestid")
	// descript := r.URL.Query().Get("description")
	// instructions := r.URL.Query().Get("instructions")
	// weightin, _ := strconv.Atoi(r.URL.Query().Get("weightin"))
	// weightout, _ := strconv.Atoi(r.URL.Query().Get("weightout"))
	// pieces, _ := strconv.Atoi(r.URL.Query().Get("pieces"))
	// hours := r.URL.Query().Get("hours")
	// checkout := r.URL.Query().Get("checkout")
	// checkin := r.URL.Query().Get("checkin")
	// sorter := r.URL.Query().Get("sorter")
	// status := r.URL.Query().Get("status")
	// manufacturerpart := r.URL.Query().Get("manufacturerpart")
	// prty := r.URL.Query().Get("priority")
	// if hours == "" {
	// 	hours = "0"
	// }

	// //Create the fields to insert
	// if sku != "" {
	// 	i = append(i, sku)
	// }
	// i = append(i, descript)
	// i = append(i, instructions)
	// if id != "" {
	// 	i = append(i, id)
	// } //ensure that the id isn't null before inserting
	// i = append(i, weightin)
	// i = append(i, weightout)
	// i = append(i, pieces)
	// i = append(i, hours)
	// i = append(i, checkout)
	// i = append(i, checkin)
	// i = append(i, sorter)
	// i = append(i, status)
	// i = append(i, manufacturerpart)
	// i = append(i, prty)
	// log.WithFields(log.Fields{"username": user.Username}).Debug("Inserting Sorting Request: ", i)
	// log.WithFields(log.Fields{"username": user.Username}).Debug(i...) //debug variables map
	// log.WithFields(log.Fields{"username": user.Username}).Debug("Running Product List")

	//Build the Query
	// if id != "" {
	// 	//Run the query if ID isn't null
	// 	newquery = "REPLACE INTO sortrequest (`sku`, `description`, `instructions`, `requestid`,`weightin`, `weightout`, `pieces`, `hours`, `checkout`, `checkint`, `sorter`,status,sku_manufacturer,prty) VALUES (REPLACE(?,' ',''),?,?,?,?,?,?,?,?,?,?,?,?,?)"
	// 	log.WithFields(log.Fields{"username": user.Username}).Debug("Query: ", newquery)
	// 	rows, err := db.Query(newquery, i...)
	// 	if err != nil {
	// 		return handleerror(err)
	// 	}
	// 	defer rows.Close()
	// } else {
	// 	//Run the query to insert a new row
	// 	newquery = "INSERT INTO sortrequest (`sku`, `description`, `instructions`, `weightin`, `weightout`, `pieces`, `hours`, `checkout`, `checkint`, `sorter`,status,sku_manufacturer,prty) VALUES (REPLACE(?,' ',''),?,?,?,?,?,?,?,?,?,?,?,?)"
	// 	log.WithFields(log.Fields{"username": user.Username}).Debug("Query: ", newquery)
	// 	rows, err := db.Query(newquery, i...)
	// 	if err != nil {
	// 		return handleerror(err)
	// 	}
	// 	defer rows.Close()
	// }

	//define variables
	var newquery string
	var values []interface{}

	data := map[string]string{
		"sku":              r.URL.Query().Get("sku"),
		"description":      r.URL.Query().Get("description"),
		"instructions":     r.URL.Query().Get("instructions"),
		"weightin":         r.URL.Query().Get("weightin"),
		"weightout":        r.URL.Query().Get("weightout"),
		"pieces":           r.URL.Query().Get("pieces"),
		"hours":            r.URL.Query().Get("hours"),
		"checkout":         r.URL.Query().Get("checkout"),
		"checkint":         r.URL.Query().Get("checkin"),
		"sorter":           r.URL.Query().Get("sorter"),
		"status":           r.URL.Query().Get("status"),
		"sku_manufacturer": r.URL.Query().Get("manufacturerpart"),
		"prty":             r.URL.Query().Get("priority"),
		"requestid":        r.URL.Query().Get("requestid"),
	}

	if data["requestid"] == "" { //if this is a new request
		newquery = "REPLACE INTO sortrequest ("
		for key, value := range data {
			if value != "" {
				newquery += "`" + key + "`,"
				values = append(values, value)
			}
		}
		newquery = newquery[:len(newquery)-1] + ") VALUES ("
		for key, value := range data {
			if value != "" {
				log.WithFields(log.Fields{"username": user.Username}).Debug("key:", key, ", value:", value)
				newquery += "?,"
			}
		}
		newquery = newquery[:len(newquery)-1] + ")"
	} else { //if updating an existing request
		newquery = "UPDATE sortrequest SET "
		for key, value := range data {
			if value == "<nil>" {
				value = "" //fix <nil> values being inserted
			}
			if value != "" {
				newquery += "`" + key + "`=?,"
				values = append(values, value)
			}
		}
		newquery = newquery[:len(newquery)-1] //get rid of the last comma
		newquery += " WHERE requestid=" + data["requestid"]
	}

	log.WithFields(log.Fields{"username": user.Username}).Debug("newquery: ", newquery)
	rows, err := db.Query(newquery, values...)
	if err != nil {
		return handleerror(err)
	}
	defer rows.Close()

	//Logging
	log.WithFields(log.Fields{"username": user.Username}).Info("Inserted Product ", data["sku"])
	message.Title = "Success"
	message.Body = "Successfully inserted row"
	message.Success = true
	return message
}

func nextorder(manufacturer string, user User) (message Message, order Order) {
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
	log.WithFields(log.Fields{"username": user.Username}).Debug(manufacturer, "-", ordernum)

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
func Reorderlist(user User) (message Message, orders []Order) {
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
		newquery := "SELECT a.sku_internal,`manufacturer_code`,`sku_manufacturer`,`product_option`,`processing_request`,`sorting_request`,`unit`,`unit_price`,`Currency`,`order_qty`,`modified`,`reorder`,`inventory_qty`,season,url_thumb,url_standard FROM `skus` a LEFT JOIN (select sku_internal FROM orderskus a left join orders b on a.ordernum = b.ordernum where status != 'Closed') b on a.sku_internal = b.sku_internal WHERE inventory_qty = 0 and reorder = 1 and b.sku_internal is null and manufacturer_code = ?"
		skurows, err := db.Query(newquery, r.Manufacturer)
		if err != nil {
			return handleerror(pingErr), orders
		}
		var skus []Product
		defer skurows.Close()
		for skurows.Next() {
			var r Product
			err := skurows.Scan(&r.SKU, &r.Manufacturer, &r.ManufacturerPart, &r.Description, &r.ProcessRequest, &r.SortingRequest, &r.Unit, &r.UnitPrice, &r.Currency, &r.Qty, &r.Modified, &r.Reorder, &r.InventoryQTY, &r.Season, &r.Image.URL_Thumb, &r.Image.URL_Standard)
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
func ProductList(limit int, r *http.Request, user User) (message Message, products []Product) {
	// Get a database handle.
	var err error

	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, message = opendb()
		return handleerror(pingErr), products
	}

	queryParams := map[string]string{
		"sku_internal":       r.URL.Query().Get("sku"),
		"manufacturer_code":  r.URL.Query().Get("manufacturer"),
		"sku_manufacturer":   r.URL.Query().Get("manufacturerpart"),
		"processing_request": r.URL.Query().Get("processrequest"),
		"sorting_request":    r.URL.Query().Get("sortingrequest"),
		"unit":               r.URL.Query().Get("unit"),
		"unit_price":         r.URL.Query().Get("unitprice"),
		"Currency":           r.URL.Query().Get("currency"),
		"order_qty":          r.URL.Query().Get("orderqty"),
		"reorder":            r.URL.Query().Get("reorder"),
		"season":             r.URL.Query().Get("season"),
	}

	var i []interface{}
	var newquery string

	newquery = "SELECT `sku_internal`,`manufacturer_code`,`sku_manufacturer`,`product_option`,`processing_request`,`sorting_request`,`unit`,`unit_price`,`Currency`,`order_qty`,`modified`,`reorder`,`inventory_qty`,season,url_standard,url_thumb,url_tiny FROM `skus` WHERE 1"

	for param, value := range queryParams {
		if value != "" {
			// if param == "sku_internal" || param == "sku_manufacturer" || param == "unit" {
			// 	value += "%"
			// } else if param == "processing_request" || param == "sorting_request" {
			// 	value = "%" + value + "%"
			// }
			i = append(i, value)
			newquery += fmt.Sprintf(" AND %s = ?", param)
		}
	}

	newquery += " order by 11 desc, 1 limit ?"
	i = append(i, limit)
	log.WithFields(log.Fields{"username": user.Username}).Debug(i...) //debug variables map
	log.WithFields(log.Fields{"username": user.Username}).Debug("Running Product List")
	log.WithFields(log.Fields{"username": user.Username}).Debug(newquery)
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

func sortrequestdeletesql(requestid int, user User) (message Message) {
	//Debug
	log.WithFields(log.Fields{"username": user.Username}).Info("Deleting order ", order, "...")

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
func ProductInsert(r *http.Request, user User) (message Message) {
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
	log.WithFields(log.Fields{"username": user.Username}).Debug("Reorder: ", reorder)

	//Build the Query
	newquery = "REPLACE INTO skus (`sku_internal`, `manufacturer_code`, `sku_manufacturer`, `processing_request`, `sorting_request`, `unit`, `unit_price`, `Currency`, `order_qty`,`product_option`,`reorder`,season) VALUES (REPLACE(?,' ',''),?,?,?,?,?,?,?,?,?,?,?)"

	//Run Query
	log.WithFields(log.Fields{"username": user.Username}).Debug(i...) //debug variables map
	log.WithFields(log.Fields{"username": user.Username}).Debug("Running Product List")
	log.WithFields(log.Fields{"username": user.Username}).Debug(newquery)
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
	log.WithFields(log.Fields{"username": user.Username}).Info("Inserted Product ", sku)
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
	var newquery string = "update orders.users set password = ? where username = ? and password = ''"
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
func userauth(username string, pass string) (user User, message Message) {
	// Get a database handle.
	var err error
	var dbpass string
	//Test Connection
	pingErr := db.Ping()
	// if pingErr != nil {
	// 	user.Role = "notfound"
	// 	return user, handleerror(pingErr)
	// }

	var r User

	//set Variables
	//Query
	var newquery string = "select password, permissions, admin, management from orders.users where username = ?"
	// log.WithFields(log.Fields{"username": user.Role}).Debug(newquery)
	rows, err := db.Query(newquery, username)
	if err != nil {
		user.Role = "notfound"
		return user, handleerror(pingErr)
	}
	defer rows.Close()
	//Pull Data
	for rows.Next() {
		err := rows.Scan(&dbpass, &r.Role, &r.Permissions.Admin, &r.Permissions.Mgmt)
		log.Debug("Role:", r.Role)
		if err != nil {
			user.Role = "notfound"
			return user, handleerror(pingErr)
		}
	}
	err = rows.Err()
	if err != nil {
		user.Role = "notfound"
		return user, handleerror(pingErr)
	}

	//If Permissions do not exist for user
	if r.Role == "" {
		message.Title = "Permission not found"
		message.Body = "Permissions not set for user. Please contact your system administrator."
		user.Role = "notfound"
		return user, message
	}

	log.Debug("Checking Permissions: ", r.Role)
	//If user has not set a password
	if dbpass == "" {
		message.Title = "Set Password"
		message.Body = "Password not set, please create password"
		user.Role = "newuser"
		return user, message
	}

	if comparePasswords(dbpass, []byte(pass)) {
		message.Title = "Success"
		message.Body = "Successfully logged in"
		message.Success = true
		// permission = "notfound"
		return r, message
	}
	message.Title = "Login Failed"
	message.Body = "Login Failed"
	user.Role = "notfound"
	return user, message
}

// Authenticate user from DB
func userdata(username string) (user User) {
	// user.Role = user
	// Get a database handle.
	var err error
	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		handleerror(pingErr)
	}
	//set Variables
	//Query
	var newquery string = "select username,usercode,permissions,admin,management,manager,sorting from orders.users where username = ?"
	// log.WithFields(log.Fields{"username": user.Role}).Debug(newquery)
	rows, err := db.Query(newquery, username)
	if err != nil {
		handleerror(err)
	}
	defer rows.Close()
	//Pull Data
	for rows.Next() {
		err := rows.Scan(&user.Username, &user.Usercode, &user.Role, &user.Permissions.Admin, &user.Permissions.Mgmt, &user.Manager, &user.Permissions.Sorting)
		if err != nil {
			handleerror(err)
		}
	}
	err = rows.Err()
	if err != nil {
		handleerror(err)
	}
	if user.Role == "" {
		user.Role = "notfound"
		return user
	}

	return user
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
