package main

import (
	"database/sql"
	"encoding/json"
	"math"
	"sort"
	"strings"
	"time"

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

type ApiResponse struct {
	Message string `json:"message"`
}

type Manufacturer struct {
	Name string `json:"name"`
}

type Product struct {
	SKU              string
	Description      *string
	Manufacturer     *string
	ManufacturerPart *string
	ProcessRequest   *string
	SortingRequest   *string
	Unit             *string
	UnitPrice        *float64
	Currency         *string
	OrderQty         *int
	Modified         *string
	Reorder          *bool
	InventoryQty     *int
	Season           *string
	Image            Image
}

type Customer struct {
	CustomerEmail   string         `json:"customer_email"`
	FirstName       sql.NullString `json:"first_name"`
	LastName        sql.NullString `json:"last_name"`
	Country         sql.NullString `json:"country"`
	RebillDay       sql.NullInt32  `json:"rebill_day"`
	RebillMonths    sql.NullInt32  `json:"rebill_months"`
	AutoRenew       bool           `json:"autorenew"`
	CratejoyStatus  sql.NullString `json:"cratejoy_status"`
	StartDate       sql.NullString `json:"start_date"` // Assuming dates are in string format, change to time.Time if using date objects
	EndDate         sql.NullString `json:"end_date"`
	MailchimpStatus sql.NullString `json:"mailchimp_status"`
}

var db *sql.DB

// Opens the databse connection for the application
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
	connectstring := os.Getenv("USER") + ":" + os.Getenv("PASS") + "@tcp(" + os.Getenv("SERVER") + ":" + os.Getenv("PORT") + ")/?parseTime=true"
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

func Dashdata(w http.ResponseWriter, r *http.Request) {
	sorter := r.FormValue("sorter")
	errorType := r.FormValue("errorType")

	//Auth User
	user := auth(w, r)

	log.WithFields(log.Fields{"username": user.Username}).Debugf("Sorter: %s", sorter)
	log.WithFields(log.Fields{"username": user.Username}).Debugf("Error Type: %s", errorType)

	// Construct the SQL query based on the selected filter values
	query := "SELECT COALESCE(sorter, 'Unknown') AS sorter, COALESCE(errortype, 'Unknown') AS errortype, COALESCE(MONTH(checkint), 0) AS month, COUNT(1) AS errorcount FROM sorterror a LEFT JOIN sortrequest b ON a.requestid = b.requestid WHERE 1"
	if sorter != "all" || errorType != "all" {
		query += " AND"
		if sorter != "all" {
			query += fmt.Sprintf(" sorter = '%s'", sorter)
		}
		if sorter != "all" && errorType != "all" {
			query += " AND"
		}
		if errorType != "all" {
			query += fmt.Sprintf(" errortype = '%s'", errorType)
		}
	}
	query += " GROUP BY sorter, errortype, month"
	log.WithFields(log.Fields{"username": user.Username}).Debugf("Executing SQL query: %s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	type Dataset struct {
		Label           string `json:"label"`
		Data            []int  `json:"data"`
		BackgroundColor string `json:"backgroundColor"`
		BorderColor     string `json:"borderColor"`
		BorderWidth     int    `json:"borderWidth"`
	}

	type Response struct {
		Labels   []string  `json:"labels"`
		Datasets []Dataset `json:"datasets"`
	}

	var response Response

	for rows.Next() {
		var sorter, errorType string
		var month, errorCount int
		err := rows.Scan(&sorter, &errorType, &month, &errorCount)
		if err != nil {
			log.Fatal(err)
		}
		if sorter == "" {
			sorter = "Unknown"
		}
		if errorType == "" {
			errorType = "Unknown"
		}

		var found bool
		for i, dataset := range response.Datasets {
			if dataset.Label == fmt.Sprintf("%s - %s", sorter, errorType) {
				response.Datasets[i].Data = append(dataset.Data, errorCount)
				found = true
				break
			}
		}

		if !found {
			var dataset Dataset
			dataset.Label = fmt.Sprintf("%s - %s", sorter, errorType)
			dataset.Data = []int{errorCount}
			dataset.BackgroundColor = "rgba(255, 99, 132, 0.2)"
			dataset.BorderColor = "rgba(255, 99, 132, 1)"
			dataset.BorderWidth = 1

			response.Datasets = append(response.Datasets, dataset)
			response.Labels = append(response.Labels, fmt.Sprintf("%s - %s", sorter, errorType))
		}
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func Efficiency(w http.ResponseWriter, r *http.Request) {

	//Auth User
	user := auth(w, r)

	//Define structs used in this query
	type Dataset struct {
		Label           string    `json:"label"`
		Data            []float64 `json:"data"`
		BackgroundColor []string  `json:"backgroundColor"`
		BorderColor     []string  `json:"borderColor"`
		BorderWidth     int       `json:"borderWidth"`
	}

	type Response struct {
		Labels   []string  `json:"labels"`
		Datasets []Dataset `json:"datasets"`
	}
	// Test connection
	pingErr := db.Ping()
	if pingErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to connect to database"))
		return
	}

	startdateStr := r.FormValue("startdate")
	enddateStr := r.FormValue("enddate")
	startdate, err := time.Parse("2006-01-02", startdateStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid start date"))
		return
	}
	enddate, err := time.Parse("2006-01-02", enddateStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid end date"))
		return
	}

	// Construct the SQL query
	query := "SELECT d.user,sum(d.items)/sum(e.hours) FROM (SELECT a.date,a.user,c.usercode,sum(b.items_total) items FROM (SELECT ordernum, station, user, DATE(scans.time) as date from orders.scans where station='pick' group by ordernum, station, user, DATE(scans.time)) a INNER JOIN (SELECT id, items_total from orders.orders) b on a.ordernum = b.id LEFT JOIN (SELECT usercode,username from orders.users) c on a.user = c.username GROUP BY a.date,a.user,c.usercode) d LEFT JOIN (SELECT DATE(clock_in) clockin,payroll_id, sum(paid_hours) hours from orders.shifts where role='Shipping' group by DATE(clock_in),payroll_id) e on d.date = e.clockin and d.usercode = e.payroll_id WHERE d.items IS NOT NULL and e.hours IS NOT NULL and d.date between ? and ? GROUP BY d.user ORDER BY 1,2;"

	log.WithFields(log.Fields{"username": user.Username}).Debugf("Executing SQL query: %s", query)
	// Execute the query
	rows, err := db.Query(query, startdate, enddate)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to execute query"))
		return
	}
	defer rows.Close()

	// Build the response
	var response Response
	var dataset Dataset
	for rows.Next() {
		var user string
		var efficiency float64
		err := rows.Scan(&user, &efficiency)
		if err != nil {
			log.Fatal(err)
		}

		var color string
		switch {
		case efficiency > 150:
			color = "rgba(75, 192, 192, 0.2)"
		case efficiency > 100:
			color = "rgba(255, 206, 86, 0.2)"
		default:
			color = "rgba(255, 99, 132, 0.2)"
		}

		dataset.Data = append(dataset.Data, efficiency)
		response.Labels = append(response.Labels, user)
		dataset.BackgroundColor = append(dataset.BackgroundColor, color)
		dataset.BorderColor = append(dataset.BorderColor, color)
		dataset.BorderWidth = 1
	}
	dataset.Label = "Efficiency"
	response.Datasets = append(response.Datasets, dataset)

	// response.Labels = []string{"Efficiency"}

	// Encode the response to JSON and write it to the HTTP response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to encode response"))
		return
	}
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
	//Auth User
	user := auth(w, r)

	fmt.Println("TEST")
	log.WithFields(log.Fields{"username": user.Username}).Debug("Inserting Error")
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
	result, err := db.Exec("REPLACE INTO purchasing.sorterror (requestid, errortype, notes,reporter) VALUES (?, ?, ?, ?)", requestid, errortype, notes, user.Username)
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

func SortErrorList(w http.ResponseWriter, r *http.Request) {

	user := auth(w, r)
	log.WithFields(log.Fields{"username": user.Username}).Debug("Generating Sorting Error List")
	type ErrorReport struct {
		ErrorType string `json:"errortype"`
		Notes     string `json:"notes"`
		RequestID uint64 `json:"requestid"`
		SKU       string `json:"sku"`
		Sorter    string `json:"sorter"`
		Checkin   string `json:"checkin"`
		Reporter  string `json:"reporter"`
	}

	// Parse query parameters for filtering
	requestIDParam := r.URL.Query().Get("requestid")
	errorTypeParam := r.URL.Query().Get("errortype")
	skuParam := r.URL.Query().Get("sku")
	sorterParam := r.URL.Query().Get("sorter")
	startDateParam := r.URL.Query().Get("startdate")
	endDateParam := r.URL.Query().Get("enddate")

	// Construct SQL query with filtering parameters
	query := "SELECT a.errortype, a.notes, a.requestid, b.sku, b.sorter, b.checkint, a.reporter FROM purchasing.sorterror a INNER JOIN purchasing.sortrequest b ON a.requestid = b.requestid"
	args := []interface{}{}
	if requestIDParam != "" {
		requestID, err := strconv.Atoi(requestIDParam)
		if err != nil {
			log.WithError(err).Error("Failed to parse request ID query parameter")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		query += " WHERE a.requestid = ?"
		args = append(args, requestID)
	}
	if errorTypeParam != "" {
		if len(args) == 0 {
			query += " WHERE"
		} else {
			query += " AND"
		}
		query += " a.errortype = ?"
		args = append(args, errorTypeParam)
	}
	if skuParam != "" {
		if len(args) == 0 {
			query += " WHERE"
		} else {
			query += " AND"
		}
		query += " b.sku = ?"
		args = append(args, skuParam)
	}
	if sorterParam != "" {
		if len(args) == 0 {
			query += " WHERE"
		} else {
			query += " AND"
		}
		query += " b.sorter = ?"
		args = append(args, sorterParam)
	}
	if startDateParam != "" {
		startDate, err := time.Parse("2006-01-02", startDateParam)
		if err != nil {
			log.WithError(err).Error("Failed to parse start date query parameter")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if len(args) == 0 {
			query += " WHERE"
		} else {
			query += " AND"
		}
		query += " b.checkint >= ?"
		args = append(args, startDate)
	}
	if endDateParam != "" {
		endDate, err := time.Parse("2006-01-02", endDateParam)
		if err != nil {
			log.WithError(err).Error("Failed to parse end date query parameter")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if len(args) == 0 {
			query += " WHERE"
		} else {
			query += " AND"
		}
		query += " b.checkint <= ?"
		args = append(args, endDate)
	}

	//Debug
	log.WithFields(log.Fields{"username": user.Username}).Debug("Query:", query)

	// Retrieve error reports from the database
	rows, err := db.Query(query, args...)
	if err != nil {
		log.WithError(err).Error("Failed to retrieve error reports")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Convert database rows to ErrorReport structs
	errorReports := []ErrorReport{}
	for rows.Next() {
		var errorReport ErrorReport
		var requestID sql.NullInt64 // Use sql.NullInt64 for the requestid field
		var reporter sql.NullString // Use sql.NullString for the reporter field
		err := rows.Scan(&errorReport.ErrorType, &errorReport.Notes, &requestID, &errorReport.SKU, &errorReport.Sorter, &errorReport.Checkin, &reporter)
		if err != nil {
			log.WithError(err).Error("Failed to scan error report")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if requestID.Valid { // Check if the requestid value is valid
			errorReport.RequestID = uint64(requestID.Int64)
		}
		if reporter.Valid { // Check if the reporter value is valid
			errorReport.Reporter = reporter.String
		}
		errorReports = append(errorReports, errorReport)
	}

	if err := rows.Err(); err != nil {
		log.WithError(err).Error("Failed to retrieve error reports")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Sort error reports by request ID in ascending order
	sort.Slice(errorReports, func(i, j int) bool {
		return errorReports[i].RequestID < errorReports[j].RequestID
	})

	// Encode error reports to JSON and write response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(errorReports); err != nil {
		log.WithError(err).Error("Failed to encode error reports")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
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
			err := skurows.Scan(&r.SKU, &r.Manufacturer, &r.ManufacturerPart, &r.Description, &r.ProcessRequest, &r.SortingRequest, &r.Unit, &r.UnitPrice, &r.Currency, &r.OrderQty, &r.Modified, &r.Reorder, &r.InventoryQty, &r.Season)
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

func UpdateUser(w http.ResponseWriter, r *http.Request) {

	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, _ = opendb()
		// return handleerror(pingErr)
	}

	user := auth(w, r)

	// Get the form values from the request
	username := r.FormValue("username")
	usercode := r.FormValue("usercode")
	role := r.FormValue("role")
	manager := r.FormValue("manager")
	log.WithFields(log.Fields{"username": user.Username}).Debug("username:", username, " usercode:", usercode, " role:", role, " manager:", manager)
	var sorting int
	if r.FormValue("sorting") == "true" {
		sorting = 1
	} else {
		sorting = 0
	}
	var management int
	if r.FormValue("management") == "true" {
		management = 1
	} else {
		management = 0
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

	// Prepare the SQL statement for inserting or updating the data
	newquery := `
		INSERT INTO orders.users (username, usercode, permissions, sorting, manager, management)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
		username = VALUES(username),
		permissions = VALUES(permissions),
		sorting = VALUES(sorting),
		manager = VALUES(manager),
		management = VALUES(management)
		`

	// Execute the SQL statement with the form values
	rows, err := db.Query(newquery, username, usercode, role, sorting, manager, management)
	defer rows.Close()

	if err != nil {
		// Handle error
		println(err)
		http.Error(w, "Failed to update user information.", http.StatusInternalServerError)
		return
	}

	// // Prepare the SQL statement for inserting the data
	// //Logging
	// log.Info("Creating Query")
	// newquery := "REPLACE INTO orders.users (username, usercode, permissions, sorting,manager,management) VALUES (?, ?, ?, ?, ?, ?)"

	// // Execute the SQL statement with the form values
	// log.Info("Executing Query")
	// rows, err := db.Query(newquery, username, usercode, role, sorting, manager, management)
	// defer rows.Close()

	// if err != nil {
	// 	// Handle error
	// 	println(err)
	// 	http.Error(w, "Failed to update user information.", http.StatusInternalServerError)
	// 	return
	// }

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
		"sku":              r.URL.Query().Get("search-sku"),
		"description":      r.URL.Query().Get("search-description"),
		"sku_manufacturer": r.URL.Query().Get("search-manufacturerpart"),
		"instructions":     r.URL.Query().Get("search-instructions"),
		"weightout":        r.URL.Query().Get("search-weightout"),
		"weightin":         r.URL.Query().Get("search-weightin"),
		"pieces":           r.URL.Query().Get("search-pieces"),
		"hours":            r.URL.Query().Get("search-hours"),
		"checkout":         r.URL.Query().Get("search-checkout"),
		"checkint":         r.URL.Query().Get("search-checkin"),
		"sorter":           r.URL.Query().Get("search-sorter"),
		"status":           r.URL.Query().Get("search-status"),
		"prty":             r.URL.Query().Get("search-priority"),
	}

	var i []interface{}
	var newquery string

	//Build the Query
	if action == "all" {
		//retrieves all records
		newquery = "SELECT requestid, sku,description,instructions,weightin,weightout,pieces,hours,checkout,checkint,COALESCE(sorter,''),status,sku_manufacturer,prty from sortrequest WHERE active=1 "
		for param, value := range queryParams {
			if value != "" {
				value = value + "%"
				i = append(i, value)
				newquery += fmt.Sprintf(" AND %s LIKE ?", param)
			}
		}
		newquery += " order by 1 desc"
	} else if action == "checkout" {
		//retrieves only records that have not been checked out yet
		newquery = "SELECT requestid, sku,description,instructions,weightin,weightout,pieces,hours,checkout,checkint,COALESCE(sorter,''),status,sku_manufacturer,prty from sortrequest WHERE active=1 AND status = 'new'"
		// if permission.Perms == "sorting" {
		// 	newquery += " AND sorter = '" + user.Username + "'"
		// }
		newquery += " order by prty desc, 1"
	} else if action == "checkin" {
		//retrieves only records that have not been checked in yet
		newquery = "SELECT requestid, sku,description,instructions,weightin,weightout,pieces,hours,checkout,checkint,COALESCE(sorter,''),status,sku_manufacturer,prty from sortrequest WHERE active=1 AND status = 'checkout'"
		if user.Role == "sorting" {
			newquery += " AND sorter = '" + user.Username + "'"
		}
		newquery += " order by 1"
	} else if action == "receiving" {
		//retrieves only records that have been checked back in
		newquery = "SELECT requestid, sku,description,instructions,weightin,weightout,pieces,hours,checkout,checkint,COALESCE(sorter,''),status,sku_manufacturer,prty from sortrequest WHERE active=1 AND status = 'checkin' order by 1 desc"
	}

	newquery += " limit 200"

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
		}
		r.Difference = a - b - (c * 0.4555)               //0.4555 is the bag weight in grams
		r.Difference = math.Round(r.Difference*100) / 100 // Round to 2 decimal places
		if a != 0 {
			r.DifferencePercent = formatAsPercent(r.Difference / a) //Get the percentage of the weight in
		}
		if r.Difference < (-0.1*a) && a != 0 {
			r.Warn = true
		}
		r.Difference = -r.Difference
		// log.Info(c)
		sortrequests = append(sortrequests, r)
	}
	return message, sortrequests
}

// Sorting Insert
func Sortinginsert(w http.ResponseWriter, r *http.Request) {
	// Test DB Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, _ = opendb()
		handleerror2(pingErr, w) // send error message to AJAX request
		return
	}

	user := auth(w, r)

	// Read the request data as JSON
	log.WithFields(log.Fields{"username": user.Username}).Debug("Decoding JSON")
	var data map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		handleerror2(err, w) // send error message to AJAX request
		return
	}

	log.WithFields(log.Fields{"username": user.Username, "request": data["requestid"]}).Debug("Fixing values")
	// Rename "sku_manufacturer" key to match the database column name
	if val, ok := data["manufacturerpart"]; ok {
		delete(data, "manufacturerpart")
		data["sku_manufacturer"] = val
	}

	// Rename "checkin" key to match the database column name
	if val, ok := data["checkin"]; ok {
		delete(data, "checkin")
		data["checkint"] = val
	}

	//fix <nil> request ID
	if data["requestid"] == "<nil>" {
		log.WithFields(log.Fields{"username": user.Username, "request": data["requestid"]}).Debug("requestid is nil")
		data["requestid"] = ""
	}

	// Remove the "difference" and "layout" fields, which are not in the database
	delete(data, "difference")
	delete(data, "layout")
	log.WithFields(log.Fields{"username": user.Username, "request": data["requestid"]}).Debug(data)

	// Check if the status is being updated to 'checkin'
	if data["status"] == "checkin" {
		// Check that hours are not blank
		if data["hours"] == nil || data["hours"] == "" {
			// Return an error message if hours are blank
			message := Message{Title: "Error", Body: "Hours cannot be blank when updating status to 'checkin'", Success: false}
			response, err := json.Marshal(message)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(response)
			return
		}
	}

	// Define variables
	var newquery string
	var values []interface{}
	var message Message

	// Construct SQL query based on request data
	log.WithFields(log.Fields{"username": user.Username, "request": data["requestid"]}).Debug("Constructing SQL")
	if data["requestid"] == nil || data["requestid"] == "" || data["requestid"] == "<nil>" { //if this is a new request
		newquery = "REPLACE INTO sortrequest ("
		for key, value := range data {
			if value == "<nil>" {
				value = "" // fix nil values being inserted
			}
			if value != nil && value != "" {
				newquery += "`" + key + "`,"
				values = append(values, value)
			}
			log.WithFields(log.Fields{"username": user.Username, "request": data["requestid"], key: value}).Debug("added to query")
		}
		newquery = newquery[:len(newquery)-1] + ") VALUES ("
		for range values {
			newquery += "?,"
		}
		newquery = newquery[:len(newquery)-1] + ")"
		// create success message and send it to AJAX request
		message = Message{Title: "Success", Body: "Successfully inserted request", Success: true}
	} else { //if updating an existing request
		newquery = "UPDATE sortrequest SET "
		for key, value := range data {
			if value == nil {
				value = "" // fix nil values being inserted
			}
			if value == "<nil>" {
				value = "" // fix nil values being inserted
			}
			if value != "" {
				newquery += "`" + key + "`=?,"
				values = append(values, value)
			}
		}
		newquery = newquery[:len(newquery)-1] //get rid of the last comma
		newquery += " WHERE requestid=?"
		values = append(values, data["requestid"])
		// create success message and send it to AJAX request
		message = Message{Title: "Success", Body: "Successfully updated request", Success: true}
	}

	log.WithFields(log.Fields{
		"requestid": data["requestid"],
		"query":     newquery,
		"values":    values,
	}).Info("Sortinginsert: received data")

	stmt, err := db.Prepare(newquery)
	if err != nil {
		handleerror2(err, w) // send error message to AJAX request
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(values...)
	if err != nil {
		handleerror2(err, w) // send error message to AJAX request
		return
	}

	// Logging
	log.WithFields(log.Fields{
		"requestid": data["requestid"],
	}).Info("Sortinginsert: request processed")

	// encode message as JSON
	response, err := json.Marshal(message)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// set content type to JSON and send response
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
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

// ReordersListHandler handles the API endpoint for retrieving reordered lists with pagination
// func ReordersListHandler(w http.ResponseWriter, r *http.Request) {
// 	// Parse the request parameters
// 	Manufacturer := r.URL.Query().Get("manufacturer")
// 	pageStr := r.URL.Query().Get("page")
// 	pageSizeStr := r.URL.Query().Get("pageSize")

// 	// Convert the parameters to integers
// 	page, err := strconv.Atoi(pageStr)
// 	if err != nil {
// 		http.Error(w, "Invalid page number", http.StatusBadRequest)
// 		return
// 	}

// 	pageSize, err := strconv.Atoi(pageSizeStr)
// 	if err != nil {
// 		http.Error(w, "Invalid pageSize", http.StatusBadRequest)
// 		return
// 	}

// 	// Call the Reorderlist function with the provided parameters
// 	//products, totalPages := ProductList2(Manufacturer, page, pageSize)

// 	// Convert the orders to JSON
// 	response := struct {
// 		Products    []Product
// 		TotalPages  int
// 		CurrentPage int
// 	}{
// 		Products:    products,
// 		TotalPages:  totalPages,
// 		CurrentPage: page,
// 	}

// 	// log.Debug("JSON:", response)

// 	jsonResponse, err := json.Marshal(response)
// 	if err != nil {
// 		http.Error(w, "Error encoding JSON response", http.StatusInternalServerError)
// 		return
// 	}

// 	// Set the appropriate headers and write the JSON response
// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusOK)
// 	w.Write(jsonResponse)
// }

// Reorders List
func Reorderlist(user User) (message Message, orders []Order) {
	// Get a database handle.
	var err error

	// Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		db, message = opendb()
		return handleerror(pingErr), orders
	}

	// Build the Query with pagination
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
		// //Build the Query for the skus in the order
		// newquery := "SELECT a.sku_internal,`manufacturer_code`,`sku_manufacturer`,`product_option`,`processing_request`,`sorting_request`,`unit`,`unit_price`,`Currency`,`order_qty`,`modified`,`reorder`,`inventory_qty`,season,url_thumb,url_standard FROM `skus` a LEFT JOIN (select sku_internal FROM orderskus a left join orders b on a.ordernum = b.ordernum where status != 'Closed') b on a.sku_internal = b.sku_internal WHERE inventory_qty = 0 and reorder = 1 and b.sku_internal is null and manufacturer_code = ?"
		// skurows, err := db.Query(newquery, r.Manufacturer)
		// if err != nil {
		// 	return handleerror(pingErr), orders
		// }
		// var skus []Product
		// defer skurows.Close()
		// for skurows.Next() {
		// 	var r Product
		// 	err := skurows.Scan(&r.SKU, &r.Manufacturer, &r.ManufacturerPart, &r.Description, &r.ProcessRequest, &r.SortingRequest, &r.Unit, &r.UnitPrice, &r.Currency, &r.OrderQty, &r.Modified, &r.Reorder, &r.InventoryQty, &r.Season, &r.Image.URL_Thumb, &r.Image.URL_Standard)
		// 	if err != nil {
		// 		return handleerror(pingErr), orders
		// 	}
		// 	skus = append(skus, r)
		// }
		// r.Products = skus
		//Append to the orders
		orders = append(orders, r)
	}

	return message, orders
}

// ListUsersAPI godoc
// @Summary List users
// @Description Get a list of users with optional role filtering
// @Tags users
// @Accept  json
// @Produce  json
// @Param role query string false "Filter users by role (sorting/manager)"
// @Success 200 {object} map[string][]User "List of users"
// @Failure 400 {object} map[string]string "Invalid request parameters"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /users [get]
// ListUsersAPI is an HTTP handler function that returns a list of users in JSON format
func ListUsersAPI(w http.ResponseWriter, r *http.Request) {
	// Extract role from query parameter if needed
	role := r.URL.Query().Get("role")

	// Test database connection
	pingErr := db.Ping()
	if pingErr != nil {
		log.Debug(pingErr)
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Database connection error"})
		return
	}

	// Build the query based on the role
	var newquery string
	if role == "sorting" {
		newquery = "SELECT username,usercode,permissions,sorting,manager,management FROM orders.users WHERE sorting=1 AND active=1"
	} else if role == "manager" {
		newquery = "SELECT username,usercode,permissions,sorting,manager,management FROM orders.users WHERE management=1 AND active=1"
	} else {
		newquery = "SELECT username,usercode,permissions,sorting,manager,management FROM orders.users WHERE active=1"
	}

	// Execute the query
	rows, err := db.Query(newquery)
	if err != nil {
		log.Debug(err)
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Error executing query"})
		return
	}
	defer rows.Close()

	var users []User
	// Fetch rows
	for rows.Next() {
		var user User
		err := rows.Scan(&user.Username, &user.Usercode, &user.Role, &user.Sorting, &user.Manager, &user.Management)
		if err != nil {
			log.Debug(err)
			respondWithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Error scanning row"})
			return
		}
		users = append(users, user)
	}

	// Handle any errors encountered during iteration
	if err = rows.Err(); err != nil {
		log.Debug(err)
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Error iterating over rows"})
		return
	}

	// Respond with the user list
	respondWithJSON(w, http.StatusOK, map[string]interface{}{"users": users})
}

// ListCustomersAPI godoc
// @Summary List customers
// @Description List customers with optional search filters and pagination
// @Tags customers
// @Accept  json
// @Produce  json
// @Param customer_email query string false "Customer Email"
// @Param first_name query string false "First Name"
// @Param last_name query string false "Last Name"
// @Param country query string false "Country"
// @Param page query int false "Page number"
// @Param limit query int false "Limit per page"
// @Success 200 {object} map[string]interface{} "Customers listed successfully"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /api/customers [get]
func ListCustomersAPI(w http.ResponseWriter, r *http.Request) {

	// Test database connection
	pingErr := db.Ping()
	if pingErr != nil {
		log.Debug(pingErr)
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Database connection error"})
		return
	}

	// Prepare query parameters from URL query
	queryParams := map[string]string{
		"customer_email": r.URL.Query().Get("customer_email"),
		"first_name":     r.URL.Query().Get("first_name"),
		"last_name":      r.URL.Query().Get("last_name"),
		"country":        r.URL.Query().Get("country"),
		"rebill_day":     r.URL.Query().Get("rebill_day"),
		"rebill_months":  r.URL.Query().Get("rebill_months"),
		"autorenew":      r.URL.Query().Get("autorenew"),
		"a.status":       r.URL.Query().Get("cratejoy_status"),
		"start_date":     r.URL.Query().Get("start_date"),
		"end_date":       r.URL.Query().Get("end_date"),
		"b.status":       r.URL.Query().Get("mailchimp_status"),
	}

	// Pagination parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10 // Default limit
	}
	offset := (page - 1) * limit

	var queryArgs []interface{}
	var queryBuilder strings.Builder
	queryBuilder.WriteString("SELECT customer_email, first_name, last_name, country, rebill_day, rebill_months, autorenew, a.status as cratejoy_status, start_date, end_date, b.status as mailchimp_status FROM customers.cratejoy_subscriptions a LEFT JOIN customers.mailchimp b on a.customer_email = b.email WHERE 1")

	for param, value := range queryParams {
		if value != "" {
			queryArgs = append(queryArgs, value)
			queryBuilder.WriteString(fmt.Sprintf(" AND %s = ?", param))
		}
	}

	// Count total records query (without LIMIT and OFFSET)
	countQuery := strings.Replace(queryBuilder.String(), "SELECT customer_email, first_name, last_name, country, rebill_day, rebill_months, autorenew, a.status as cratejoy_status, start_date, end_date, b.status as mailchimp_status", "SELECT COUNT(*)", 1)
	log.WithFields(log.Fields{"countQuery": countQuery, "args": queryArgs}).Debug("Executing count query")

	// Execute count query
	var totalRecords int
	err := db.QueryRow(countQuery, queryArgs...).Scan(&totalRecords)
	if err != nil {
		log.Debug(err)
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Error executing count query"})
		return
	}
	totalPages := (totalRecords + limit - 1) / limit // Calculate total pages

	// Add pagination to the main query
	queryArgs = append(queryArgs, limit, offset)
	query := queryBuilder.String() + " LIMIT ? OFFSET ?"
	log.WithFields(log.Fields{"query": query, "args": queryArgs}).Debug("Executing query")

	rows, err := db.Query(query, queryArgs...)
	if err != nil {
		log.Debug(err)
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Error executing query"})
		return
	}
	defer rows.Close()

	var customers []Customer
	// Fetch rows
	for rows.Next() {
		var customer Customer
		err := rows.Scan(&customer.CustomerEmail, &customer.FirstName, &customer.LastName, &customer.Country, &customer.RebillDay, &customer.RebillMonths, &customer.AutoRenew, &customer.CratejoyStatus, &customer.StartDate, &customer.EndDate, &customer.MailchimpStatus)
		if err != nil {
			log.Error(err)
			respondWithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Error scanning row"})
			return
		}
		customers = append(customers, customer)
	}

	// Handle any errors encountered during iteration
	if err = rows.Err(); err != nil {
		log.Debug(err)
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Error iterating over rows"})
		return
	}

	// Respond with the customer list and pagination info
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"customers":    customers,
		"currentPage":  page,
		"perPage":      limit,
		"totalPages":   totalPages,
		"totalRecords": totalRecords,
	})
}

// UserDeleteAPI godoc
// @Summary Delete a user
// @Description Deletes a user with the given usercode
// @Tags users
// @Accept  json
// @Produce  json
// @Param usercode body string true "Usercode of the user to delete"
// @Success 200 {object} map[string]string{"message": "User successfully deleted"}
// @Failure 400 {object} map[string]string{"error": "Bad Request: Usercode is required"}
// @Failure 404 {object} map[string]string{"error": "User not found"}
// @Failure 500 {object} map[string]string{"error": "Internal Server Error"}
// @Router /api/userdelete [post]
// UserDeleteAPI handles the deletion of a user
func UserDeleteAPI(w http.ResponseWriter, r *http.Request) {
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error decoding user data")
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Internal Server Error: %v", err)})
		return
	}

	// SQL DELETE statement
	query := `UPDATE users SET active=0 WHERE usercode=?`
	_, err = db.Exec(query, user.Usercode)
	if err != nil {
		log.WithFields(log.Fields{"usercode": user.Usercode, "error": err}).Error("Error executing delete query")
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Internal Server Error: %v", err)})
		return
	}

	// Respond with success message
	respondWithJSON(w, http.StatusOK, map[string]string{"message": "User successfully deleted"})
}

// ListManufacturers godoc
// @Summary List manufacturers
// @Description Retrieves a list of manufacturers from the database
// @Tags manufacturers
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @param Authorization header string true "Authorization"
// @Success 200 {array} Manufacturer "List of manufacturers"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/manufacturers [get]
// Manufacturer List API
func ListManufacturers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT name FROM purchasing.manufacturers WHERE 1")
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error executing query")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var manufacturers []Manufacturer
	for rows.Next() {
		var m Manufacturer
		if err := rows.Scan(&m.Name); err != nil {
			log.WithFields(log.Fields{"error": err}).Error("Error scanning row")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		manufacturers = append(manufacturers, m)
	}

	// Handle any error encountered during iteration
	if err = rows.Err(); err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error iterating over rows")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Set header and encode the result into JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(manufacturers); err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error encoding JSON")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// ProductList godoc
// @Summary List products
// @Description Retrieves a list of products, with optional filtering parameters
// @Tags products
// @Accept  json
// @Produce  json
// @Security X-API-Key
// @Param sku_internal query string false "Filter by internal SKU"
// @Param manufacturer_code query string false "Filter by manufacturer code"
// @Param sku_manufacturer query string false "Filter by manufacturer SKU"
// @Param processing_request query string false "Filter by processing request"
// @Param sorting_request query string false "Filter by sorting request"
// @Param unit query string false "Filter by unit"
// @Param unit_price query string false "Filter by unit price"
// @Param Currency query string false "Filter by currency"
// @Param order_qty query string false "Filter by order quantity"
// @Param reorder query string false "Filter by reorder status"
// @Param season query string false "Filter by season"
// @Param limit query int false "Limit number of results"
// @Success 200 {array} Product "List of products"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/products [get]
// ProductListAPI is an HTTP handler function that returns product list in JSON format
func ProductList(w http.ResponseWriter, r *http.Request) {
	log.Debug("Entering ProductListAPI")

	// Check database connection
	pingErr := db.Ping()
	if pingErr != nil {
		log.WithFields(log.Fields{"error": pingErr}).Error("Database connection error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Extract query parameters from request
	queryParams := map[string]string{
		"sku_internal":       r.URL.Query().Get("sku"),
		"manufacturer_code":  r.URL.Query().Get("manufacturer"),
		"sku_manufacturer":   r.URL.Query().Get("manufacturerpart"),
		"product_option":     r.URL.Query().Get("description"),
		"processing_request": r.URL.Query().Get("processrequest"),
		"unit":               r.URL.Query().Get("unit"),
		"unit_price":         r.URL.Query().Get("unitprice"),
		"currency":           r.URL.Query().Get("currency"),
		"order_qty":          r.URL.Query().Get("orderqty"),
		// "reorder":            r.URL.Query().Get("reorder"),
		"season": r.URL.Query().Get("season"),
	}

	// Pagination parameters
	currentPage := 1
	if pageParam := r.URL.Query().Get("page"); pageParam != "" {
		if newPage, err := strconv.Atoi(pageParam); err == nil && newPage > 0 {
			currentPage = newPage
		}
	}

	// Default limit
	limit := 100
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if newLimit, err := strconv.Atoi(limitParam); err == nil && newLimit > 0 {
			limit = newLimit
		}
	}

	// Calculate offset for SQL query
	offset := (currentPage - 1) * limit

	// Debug
	log.Debug("Search Params: ", queryParams)

	// Build the SQL query dynamically
	var queryArgs []interface{}
	var queryBuilder strings.Builder

	queryBuilder.WriteString("SELECT `sku_internal`,`manufacturer_code`,`sku_manufacturer`,`product_option`,`processing_request`,`sorting_request`,`unit`,`unit_price`,`Currency`,`order_qty`,`modified`,`reorder`,`inventory_qty`,season,url_standard,url_thumb,url_tiny FROM purchasing.skus WHERE 1")

	for param, value := range queryParams {
		if value != "" && param != "limit" { // Exclude 'limit' from filtering
			queryBuilder.WriteString(fmt.Sprintf(" AND %s LIKE ?", param))
			queryArgs = append(queryArgs, "%"+value+"%")
		}
	}

	// Append order by and limit
	queryBuilder.WriteString(" ORDER BY modified DESC, sku_internal LIMIT ? OFFSET ?")
	queryArgs = append(queryArgs, limit, offset)

	log.WithFields(log.Fields{"query": queryBuilder.String(), "args": queryArgs}).Debug("Executing query")

	rows, err := db.Query(queryBuilder.String(), queryArgs...)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error executing query")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.SKU, &p.Manufacturer, &p.ManufacturerPart, &p.Description, &p.ProcessRequest, &p.SortingRequest, &p.Unit, &p.UnitPrice, &p.Currency, &p.OrderQty, &p.Modified, &p.Reorder, &p.InventoryQty, &p.Season, &p.Image.URL_Standard, &p.Image.URL_Thumb, &p.Image.URL_Tiny); err != nil {
			log.WithFields(log.Fields{"error": err}).Error("Error scanning row")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		products = append(products, p)
	}

	// Calculate total number of records and total pages
	var totalRecords int
	err = db.QueryRow("SELECT COUNT(*) FROM purchasing.skus").Scan(&totalRecords)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error getting total number of records")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	totalPages := (totalRecords + limit - 1) / limit

	// Encode and send JSON response with pagination details
	responseData := map[string]interface{}{
		"currentPage":  currentPage,
		"totalPages":   totalPages,
		"totalRecords": totalRecords,
		"perPage":      limit,
		"products":     products,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(responseData); err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error encoding JSON")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	log.Debug("Exiting ProductListAPI")
}

// InsertProduct godoc
// @Summary Insert new product
// @Description Adds a new product to the database
// @Tags products
// @Accept  json
// @Produce  json
// @Security X-API-Key
// @Param product body Product true "Product to add"
// @Router /api/productinsert [post]
func InsertProduct(w http.ResponseWriter, r *http.Request) {
	var p Product
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error decoding product data")
		errMsg := fmt.Sprintf("Internal Server Error: %v", err)
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": errMsg})
		return
	}

	// Validate that SKU is provided
	if p.SKU == "" {
		log.Error("SKU is required for product insertion")
		respondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Bad Request: SKU is required"})
		return
	}

	// SQL INSERT statement
	query := `REPLACE INTO skus (sku_internal, sku_manufacturer,  product_option, manufacturer_code, processing_request, unit, unit_price, order_qty, reorder, season, inventory_qty, Currency) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,?)`
	_, err = db.Exec(query, p.SKU, p.ManufacturerPart, p.Description, p.Manufacturer, p.ProcessRequest, p.Unit, p.UnitPrice, p.OrderQty, p.Reorder, p.Season, p.InventoryQty, p.Currency)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error executing insert query")
		errMsg := fmt.Sprintf("Internal Server Error: %v", err)
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": errMsg})
		return
	}

	//update quantity and image urls
	qty(p.SKU)

	// Respond with success message
	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Product successfully inserted"})
}

// DeleteProduct godoc
// @Summary Delete a product
// @Description Deletes a product from the database based on its SKU
// @Tags products
// @Accept  json
// @Produce  json
// @Security X-API-Key
// @Param sku body string true "SKU of the product to delete"
// @Router /api/productdelete [post]
func DeleteProduct(w http.ResponseWriter, r *http.Request) {
	log.Info("Received request to delete product")

	var requestBody map[string]string
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error decoding request body")
		errMsg := fmt.Sprintf("Internal Server Error: %v", err)
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": errMsg})
		return
	}

	sku, ok := requestBody["sku"]
	if !ok || sku == "" {
		log.Error("SKU is missing in delete request")
		respondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "SKU is required"})
		return
	}

	log.WithFields(log.Fields{"SKU": sku}).Info("Parsed delete request")

	// SQL DELETE statement
	query := `DELETE FROM skus WHERE sku_internal = ?`
	result, err := db.Exec(query, sku)
	if err != nil {
		log.WithFields(log.Fields{"SKU": sku, "error": err}).Error("Error executing delete query")
		errMsg := fmt.Sprintf("Internal Server Error: %v", err)
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": errMsg})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.WithFields(log.Fields{"SKU": sku, "error": err}).Error("Error getting rows affected")
		errMsg := fmt.Sprintf("Internal Server Error: %v", err)
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": errMsg})
		return
	}

	if rowsAffected == 0 {
		log.WithFields(log.Fields{"SKU": sku}).Warn("Product not found for deletion")
		respondWithJSON(w, http.StatusNotFound, map[string]string{"error": "Product not found"})
		return
	}

	log.WithFields(log.Fields{"SKU": sku, "rowsAffected": rowsAffected}).Info("Product deleted successfully")
	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Product successfully deleted"})
}

// UpdateProduct godoc
// @Summary Update an existing product
// @Description Updates fields of a product in the database with the given SKU; only provided fields are updated
// @Tags products
// @Accept  json
// @Produce  json
// @Security X-API-Key
// @Param sku path string true "SKU of the product"
// @Param manufacturer_part body string false "Manufacturer Part of the product"
// @Param description body string false "Description of the product"
// @Param manufacturer body string false "Manufacturer of the product"
// @Param process_request body string false "Processing request of the product"
// @Param unit body string false "Unit of the product"
// @Param unit_price body number false "Unit Price of the product"
// @Param order_qty body integer false "Order Quantity of the product"
// @Param season body string false "Season of the product"
// @Param inventory_qty body integer false "Inventory Quantity of the product"
// @Param currency body string false "Currency of the product"
// @Success 200 {object} ApiResponse "Product successfully updated"
// @Failure 400 {object} ApiResponse "Bad Request: SKU is required"
// @Failure 404 {object} ApiResponse "Product not found"
// @Failure 500 {object} ApiResponse "Internal Server Error"
// @Router /api/productupdate [post]
func UpdateProduct(w http.ResponseWriter, r *http.Request) {
	var p Product
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error decoding product data")
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Internal Server Error: %v", err)})
		return
	}

	if p.SKU == "" {
		log.Error("SKU is required for product update")
		respondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Bad Request: SKU is required"})
		return
	}

	var queryArgs []interface{}
	var queryBuilder strings.Builder
	queryBuilder.WriteString("UPDATE skus SET ")

	// Dynamically build query based on provided fields
	if p.ManufacturerPart != nil {
		queryBuilder.WriteString("sku_manufacturer=?, ")
		queryArgs = append(queryArgs, *p.ManufacturerPart)
	}
	if p.Description != nil {
		queryBuilder.WriteString("product_option=?, ")
		queryArgs = append(queryArgs, *p.Description)
	}
	if p.Manufacturer != nil {
		queryBuilder.WriteString("manufacturer_code=?, ")
		queryArgs = append(queryArgs, *p.Manufacturer)
	}
	if p.ProcessRequest != nil {
		queryBuilder.WriteString("processing_request=?, ")
		queryArgs = append(queryArgs, *p.ProcessRequest)
	}
	if p.SortingRequest != nil {
		queryBuilder.WriteString("sorting_request=?, ")
		queryArgs = append(queryArgs, *p.SortingRequest)
	}
	if p.Unit != nil {
		queryBuilder.WriteString("unit=?, ")
		queryArgs = append(queryArgs, *p.Unit)
	}
	if p.UnitPrice != nil {
		queryBuilder.WriteString("unit_price=?, ")
		queryArgs = append(queryArgs, *p.UnitPrice)
	}
	if p.OrderQty != nil {
		queryBuilder.WriteString("order_qty=?, ")
		queryArgs = append(queryArgs, *p.OrderQty)
	}
	if p.Reorder != nil {
		queryBuilder.WriteString("reorder=?, ")
		queryArgs = append(queryArgs, *p.Reorder)
	}
	if p.Season != nil {
		queryBuilder.WriteString("season=?, ")
		queryArgs = append(queryArgs, *p.Season)
	}
	if p.InventoryQty != nil {
		queryBuilder.WriteString("inventory_qty=?, ")
		queryArgs = append(queryArgs, *p.InventoryQty)
	}
	if p.Currency != nil {
		queryBuilder.WriteString("Currency=?, ")
		queryArgs = append(queryArgs, *p.Currency)
	}

	// Remove trailing comma and space
	query := strings.TrimSuffix(queryBuilder.String(), ", ")
	query += " WHERE sku_internal=?"
	queryArgs = append(queryArgs, p.SKU)

	_, err = db.Exec(query, queryArgs...)
	if err != nil {
		log.WithFields(log.Fields{"SKU": p.SKU, "error": err}).Error("Error executing update query")
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Internal Server Error: %v", err)})
		return
	}

	//return JSON response
	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Product successfully updated"})
}

// ListSortRequestsAPI godoc
// @Summary List sort requests
// @Description List sort requests with optional search filters
// @Tags sortrequests
// @Accept  json
// @Produce  json
// @Param sku query string false "SKU"
// @Param description query string false "Description"
// @Param manufacturer_part query string false "Manufacturer Part"
// @Param instructions query string false "Instructions"
// @Param weightout query string false "Weight Out"
// @Param weightin query string false "Weight In"
// @Param pieces query string false "Pieces"
// @Param hours query string false "Hours"
// @Param checkout query string false "Checkout"
// @Param checkin query string false "Checkin"
// @Param sorter query string false "Sorter"
// @Param status query string false "Status"
// @Param priority query string false "Priority"
// @Success 200 {object} map[string]interface{} "Sort requests listed successfully"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /api/sortrequests [get]
func ListSortRequestsAPI(w http.ResponseWriter, r *http.Request) {
	// Set up database connection (assuming db is a global *sql.DB)
	// db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/dbname")
	// if err != nil {
	// 	log.Println("Error connecting to the database: ", err)
	// 	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	// 	return
	// }
	// defer db.Close()

	// Pagination parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10 // Default limit
	}

	// Calculate offset
	offset := (page - 1) * limit

	//Gather Search Parameters
	queryParams := map[string]string{
		"sku":              r.URL.Query().Get("search-sku"),
		"description":      r.URL.Query().Get("search-description"),
		"sku_manufacturer": r.URL.Query().Get("search-manufacturerpart"),
		"instructions":     r.URL.Query().Get("search-instructions"),
		"weightout":        r.URL.Query().Get("search-weightout"),
		"weightin":         r.URL.Query().Get("search-weightin"),
		"pieces":           r.URL.Query().Get("search-pieces"),
		"hours":            r.URL.Query().Get("search-hours"),
		"checkout":         r.URL.Query().Get("search-checkout"),
		"checkint":         r.URL.Query().Get("search-checkin"),
		"sorter":           r.URL.Query().Get("search-sorter"),
		"status":           r.URL.Query().Get("search-status"),
		"prty":             r.URL.Query().Get("search-priority"),
	}

	// Construct SQL query
	var queryArgs []interface{}
	var queryBuilder strings.Builder
	queryBuilder.WriteString("SELECT requestid, sku,description,instructions,weightin,weightout,pieces,hours,checkout,checkint,COALESCE(sorter,''),status,sku_manufacturer,prty from purchasing.sortrequest WHERE active=1 ")
	for param, value := range queryParams {
		if value != "" {
			value = value + "%"
			queryArgs = append(queryArgs, value)
			queryBuilder.WriteString(fmt.Sprintf(" AND %s LIKE ?", param))
		}
	}

	// Count total records query (without LIMIT and OFFSET)
	countQuery := strings.Replace(queryBuilder.String(), "SELECT requestid, sku,description,instructions,weightin,weightout,pieces,hours,checkout,checkint,COALESCE(sorter,''),status,sku_manufacturer,prty", "SELECT COUNT(*)", 1)
	log.WithFields(log.Fields{"countQuery": countQuery, "args": queryArgs}).Debug("Executing count query")

	// Execute count query
	var totalRecords int
	err := db.QueryRow(countQuery, queryArgs...).Scan(&totalRecords)
	if err != nil {
		log.Debug(err)
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Error executing count query"})
		return
	}
	totalPages := (totalRecords + limit - 1) / limit // Calculate total pages

	// Append limit and offset to the query
	queryArgs = append(queryArgs, limit, offset)
	query := queryBuilder.String() + " LIMIT ? OFFSET ?"

	// Execute query
	rows, err := db.Query(query, queryArgs...)
	if err != nil {
		log.Debug(err)
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Error executing query"})
		return
	}
	defer rows.Close()

	// Process rows
	var sortRequests []SortRequest
	//Pull Data
	for rows.Next() {
		var r SortRequest
		err := rows.Scan(&r.ID, &r.SKU, &r.Description, &r.Instructions, &r.Weightin, &r.Weightout, &r.Pieces, &r.Hours, &r.Checkout, &r.Checkin, &r.Sorter, &r.Status, &r.ManufacturerPart, &r.Priority)
		if err != nil {
			log.Error(err)
			respondWithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Error scanning row"})
			return
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
		}
		r.Difference = a - b - (c * 0.4555)               //0.4555 is the bag weight in grams
		r.Difference = math.Round(r.Difference*100) / 100 // Round to 2 decimal places
		if a != 0 {
			r.DifferencePercent = formatAsPercent(r.Difference / a) //Get the percentage of the weight in
		}
		if r.Difference < (-0.1*a) && a != 0 {
			r.Warn = true
		}
		r.Difference = -r.Difference
		// log.Info(c)
		sortRequests = append(sortRequests, r)
	}

	// Handle any errors encountered during iteration
	if err = rows.Err(); err != nil {
		log.Println("Error iterating over rows: ", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Send JSON response with pagination details
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"sortRequests": sortRequests,
		"currentPage":  page,
		"totalPages":   totalPages,
		"totalRecords": totalRecords,
	})
}

// respondWithJSON sends a JSON response to the client.
// It logs and handles errors that occur during JSON marshaling or writing to the response.
func respondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	// Convert the payload to a JSON string
	response, err := json.Marshal(payload)
	if err != nil {
		// Log the error with the payload that caused it
		log.WithFields(log.Fields{
			"error":   err,
			"payload": payload,
		}).Error("Error marshalling JSON")

		// Respond with an Internal Server Error status and message
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error: Error marshalling JSON"))
		return
	}

	// Set the Content-Type of the response to application/json
	w.Header().Set("Content-Type", "application/json")

	// Write the status code to the response
	w.WriteHeader(statusCode)

	// Write the JSON response
	_, writeErr := w.Write(response)
	if writeErr != nil {
		// Log the error that occurred while writing the response
		log.WithFields(log.Fields{
			"error":   writeErr,
			"payload": payload,
		}).Error("Error writing JSON response")
	}

	// Log the successful response
	log.WithFields(log.Fields{
		"statusCode": statusCode,
		"response":   string(response),
	}).Debug("JSON response sent successfully")
}

// // Product List
// func ProductList(limit int, r *http.Request, user User) (message Message, products []Product) {
// 	// Get a database handle.
// 	var err error

// 	//Test Connection
// 	pingErr := db.Ping()
// 	if pingErr != nil {
// 		db, message = opendb()
// 		return handleerror(pingErr), products
// 	}

// 	queryParams := map[string]string{
// 		"sku_internal":       r.URL.Query().Get("sku"),
// 		"manufacturer_code":  r.URL.Query().Get("manufacturer"),
// 		"sku_manufacturer":   r.URL.Query().Get("manufacturerpart"),
// 		"processing_request": r.URL.Query().Get("processrequest"),
// 		"sorting_request":    r.URL.Query().Get("sortingrequest"),
// 		"unit":               r.URL.Query().Get("unit"),
// 		"unit_price":         r.URL.Query().Get("unitprice"),
// 		"Currency":           r.URL.Query().Get("currency"),
// 		"order_qty":          r.URL.Query().Get("orderqty"),
// 		"reorder":            r.URL.Query().Get("reorder"),
// 		"season":             r.URL.Query().Get("season"),
// 	}

// 	var i []interface{}
// 	var newquery string

// 	newquery = "SELECT `sku_internal`,`manufacturer_code`,`sku_manufacturer`,`product_option`,`processing_request`,`sorting_request`,`unit`,`unit_price`,`Currency`,`order_qty`,`modified`,`reorder`,`inventory_qty`,season,url_standard,url_thumb,url_tiny FROM `skus` WHERE 1"

// 	for param, value := range queryParams {
// 		if value != "" {
// 			// if param == "sku_internal" || param == "sku_manufacturer" || param == "unit" {
// 			// 	value += "%"
// 			// } else if param == "processing_request" || param == "sorting_request" {
// 			// 	value = "%" + value + "%"
// 			// }
// 			i = append(i, value)
// 			newquery += fmt.Sprintf(" AND %s = ?", param)
// 		}
// 	}

// 	newquery += " order by 11 desc, 1 limit ?"
// 	i = append(i, limit)
// 	log.WithFields(log.Fields{"username": user.Username}).Debug(i...) //debug variables map
// 	log.WithFields(log.Fields{"username": user.Username}).Debug("Running Product List")
// 	log.WithFields(log.Fields{"username": user.Username}).Debug(newquery)
// 	rows, err := db.Query(newquery, i...)
// 	if err != nil {
// 		return handleerror(err), products
// 	}
// 	defer rows.Close()

// 	//Pull Data
// 	for rows.Next() {
// 		var r Product
// 		err := rows.Scan(&r.SKU, &r.Manufacturer, &r.ManufacturerPart, &r.Description, &r.ProcessRequest, &r.SortingRequest, &r.Unit, &r.UnitPrice, &r.Currency, &r.OrderQty, &r.Modified, &r.Reorder, &r.InventoryQty, &r.Season, &r.Image.URL_Standard, &r.Image.URL_Thumb, &r.Image.URL_Tiny)
// 		if err != nil {
// 			return handleerror(err), products
// 		}
// 		products = append(products, r)
// 	}

// 	return message, products
// }

func sortrequestdeletesql(requestid int, user User) (message Message) {
	//Debug
	log.WithFields(log.Fields{"username": user.Username}).Info("Deleting order ", "...")

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

// // Product Insert
// func ProductInsert(r *http.Request, user User) (message Message) {
// 	// Get a database handle.
// 	var err error

// 	//Test Connection
// 	pingErr := db.Ping()
// 	if pingErr != nil {
// 		db, message = opendb()
// 		return handleerror(pingErr)
// 	}

// 	//Define Variables
// 	var i []interface{}
// 	var newquery string

// 	sku := r.URL.Query().Get("sku")
// 	descript := r.URL.Query().Get("description")
// 	manufacturer := r.URL.Query().Get("manufacturer")
// 	manufacturerpart := r.URL.Query().Get("manufacturerpart")
// 	processrequest := r.URL.Query().Get("processrequest")
// 	sortingrequest := r.URL.Query().Get("sortingrequest")
// 	unit := r.URL.Query().Get("unit")
// 	unitprice := r.URL.Query().Get("unitprice")
// 	currency := r.URL.Query().Get("currency")
// 	orderqty := r.URL.Query().Get("orderqty")
// 	reorder := r.URL.Query().Get("reorder")
// 	season := r.URL.Query().Get("season")

// 	//ensure that there are no null numerical values
// 	if unitprice == "" {
// 		unitprice = "0"
// 	}
// 	if orderqty == "" {
// 		orderqty = "0"
// 	}

// 	//Create the fields to insert
// 	i = append(i, sku)
// 	i = append(i, manufacturer)
// 	i = append(i, manufacturerpart)
// 	i = append(i, processrequest)
// 	i = append(i, sortingrequest)
// 	i = append(i, unit)
// 	i = append(i, unitprice)
// 	i = append(i, currency)
// 	i = append(i, orderqty)
// 	i = append(i, descript)
// 	if reorder == "yes" {
// 		i = append(i, 1)
// 	} else {
// 		i = append(i, 0)
// 	}
// 	i = append(i, season)
// 	log.WithFields(log.Fields{"username": user.Username}).Debug("Reorder: ", reorder)

// 	//Build the Query
// 	newquery = "REPLACE INTO skus (`sku_internal`, `manufacturer_code`, `sku_manufacturer`, `processing_request`, `sorting_request`, `unit`, `unit_price`, `Currency`, `order_qty`,`product_option`,`reorder`,season) VALUES (REPLACE(?,' ',''),?,?,?,?,?,?,?,?,?,?,?)"

// 	//Run Query
// 	log.WithFields(log.Fields{"username": user.Username}).Debug(i...) //debug variables map
// 	log.WithFields(log.Fields{"username": user.Username}).Debug("Running Product List")
// 	log.WithFields(log.Fields{"username": user.Username}).Debug(newquery)
// 	rows, err := db.Query(newquery, i...)
// 	if err != nil {
// 		return handleerror(err)
// 	}
// 	defer rows.Close()

// 	//Pull Data
// 	for rows.Next() {
// 		var r Product
// 		err := rows.Scan(&r.SKU, &r.Manufacturer, &r.ManufacturerPart, &r.Description, &r.ProcessRequest, &r.SortingRequest, &r.Unit, &r.UnitPrice, &r.Currency, &r.OrderQty)
// 		if err != nil {
// 			return handleerror(err)
// 		}
// 	}

// 	//add image and qty to new row
// 	qty(sku)

// 	//Logging
// 	log.WithFields(log.Fields{"username": user.Username}).Info("Inserted Product ", sku)
// 	message.Title = "Success"
// 	message.Body = "Successfully inserted row"
// 	message.Success = true
// 	return message
// }

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
