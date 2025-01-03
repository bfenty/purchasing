package main

import (
	"database/sql"
	"encoding/json"
	"sort"
	"time"

	// "log"

	"net/http"
	"purchasing/api"
	"purchasing/config"
	"purchasing/handler"
	"purchasing/models"
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

func SortErrorUpdate(w http.ResponseWriter, r *http.Request) {
	// Assuming 'auth' function checks for authentication and returns a User object
	user := api.Auth(w, r)

	log.WithFields(log.Fields{"username": user.Username}).Debug("Attempting to insert sort error")

	// Parse the form data
	if err := r.ParseForm(); err != nil {
		log.WithError(err).Error("Error parsing form data")
		handler.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Error parsing form data"})
		return
	}

	// Extract form data
	requestID := r.FormValue("requestid")
	sorter := r.FormValue("sorter")           // Assuming sorter is not used in the insert query
	description := r.FormValue("description") // Assuming description is for logging or response
	errorType := r.FormValue("errortype")
	notes := r.FormValue("notes")

	// Perform the database insert
	result, err := config.DB.Exec("REPLACE INTO purchasing.sorterror (requestid, errortype, notes, reporter) VALUES (?, ?, ?, ?)",
		requestID, errorType, notes, user.Username)
	if err != nil {
		log.WithError(err).Error("Error inserting sort error into database")
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Error inserting sort error into database"})
		return
	}

	errorID, err := result.LastInsertId()
	if err != nil {
		log.WithError(err).Error("Error getting last inserted error ID")
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Error getting last inserted error ID"})
		return
	}

	// Prepare and send the response
	response := map[string]interface{}{
		"message":   "Sort error reported successfully",
		"errorID":   errorID,
		"requestID": requestID,
		// Including sorter and description in response for completeness
		"sorter":      sorter,
		"description": description,
		"errorType":   errorType,
		"notes":       notes,
	}
	handler.RespondWithJSON(w, http.StatusOK, response)

	log.WithFields(log.Fields{
		"errorID":     errorID,
		"requestID":   requestID,
		"sorter":      sorter,
		"description": description,
		"errorType":   errorType,
		"notes":       notes,
	}).Info("Sort error reported successfully")
}

func SortErrorList(w http.ResponseWriter, r *http.Request) {

	user := api.Auth(w, r)
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
	rows, err := config.DB.Query(query, args...)
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
	rows, err := config.DB.Query("SELECT errorid, errortype, notes FROM sorterror WHERE requestid = ?", requestID)
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

func ProductExistSQL(sku string) (exists string, message models.Message) {
	log.Info("SKU: ", sku)
	//Test Connection
	pingErr := config.DB.Ping()
	if pingErr != nil {
		config.DB, message = config.Opendb()
		return exists, handler.Handleerror(pingErr)
	}

	// sku = "TEST"
	newquery := "SELECT COUNT(*) from skus where sku_internal = ?"
	var count int
	err := config.DB.QueryRow(newquery, sku).Scan(&count)
	if err != nil {
		handler.Handleerror(err)
	}
	log.Info("Count: ", count)

	if count == 0 {
		exists = "FALSE"
	} else {
		exists = "TRUE"
	}

	return exists, message
}

func orderdeletesql(order int, user models.User) (message models.Message) {
	//Debug
	log.WithFields(log.Fields{"username": user.Username}).Info("Deleting order ", order, "...")

	//Test Connection
	pingErr := config.DB.Ping()
	if pingErr != nil {
		config.DB, message = config.Opendb()
		return handler.Handleerror(pingErr)
	}

	//Build the Query
	newquery := "DELETE FROM `orderskus` WHERE ordernum = ?"
	rows, err := config.DB.Query(newquery, order)
	rows.Close()
	if err != nil {
		return handler.Handleerror(err)
	}

	//Build the Query
	newquery = "DELETE FROM `orders` WHERE ordernum = ?"
	rows, err = config.DB.Query(newquery, order)
	rows.Close()
	if err != nil {
		return handler.Handleerror(err)
	}

	message.Success = true
	message.Title = "Success"
	message.Body = "Successfully deleted order " + strconv.Itoa(order)
	return message
}

func orderskuadd(order int, sku string, user models.User) (message models.Message) {
	//Debug
	log.WithFields(log.Fields{"username": user.Username}).Info("Inserting SKU/Order: ", sku, "/", order)

	//Test Connection
	pingErr := config.DB.Ping()
	if pingErr != nil {
		config.DB, message = config.Opendb()
		return handler.Handleerror(pingErr)
	}
	//Build the Query
	newquery := "REPLACE INTO `orderskus`(`ordernum`, `sku_internal`) VALUES (?,?)"

	rows, err := config.DB.Query(newquery, order, sku)
	rows.Close()
	if err != nil {
		return handler.Handleerror(err)
	}

	message.Body = "Successfully inserted SKU " + sku
	message.Success = true
	return message
}

func orderlookup(ordernum int, user models.User) (message models.Message, orders []models.Order) {
	//Debug
	log.WithFields(log.Fields{"username": user.Username}).Debug("Getting Order: ", strconv.Itoa(ordernum))

	//Test Connection
	pingErr := config.DB.Ping()
	if pingErr != nil {
		config.DB, message = config.Opendb()
		return handler.Handleerror(pingErr), orders
	}
	//Build the Query
	newquery := "SELECT ordernum,trackingnum,comments,manufacturer,status FROM `orders` WHERE ordernum = ?"

	orderrows, err := config.DB.Query(newquery, ordernum)
	if err != nil {
		return handler.Handleerror(pingErr), orders
	}
	defer orderrows.Close()
	log.WithFields(log.Fields{"username": user.Username}).Debug("Orderrows: ", orderrows)
	//Pull Data
	for orderrows.Next() {
		var r models.Order
		err := orderrows.Scan(&r.Ordernum, &r.Tracking, &r.Comments, &r.Manufacturer, &r.Status)
		if err != nil {
			return handler.Handleerror(pingErr), orders
		}
		//Build the Query for the skus in the order
		newquery := "SELECT a.sku_internal,`manufacturer_code`,`sku_manufacturer`,`product_option`,`processing_request`,`sorting_request`,`unit`,`unit_price`,`Currency`,`order_qty`,`modified`,`reorder`,`inventory_qty`,season FROM orderskus a left join skus b on a.sku_internal = b.sku_internal WHERE a.ordernum = ?"
		skurows, err := config.DB.Query(newquery, r.Ordernum)
		if err != nil {
			return handler.Handleerror(pingErr), orders
		}
		log.WithFields(log.Fields{"username": user.Username}).Debug("SKUrows: ", skurows)
		var skus []models.Product
		defer skurows.Close()
		for skurows.Next() {
			var r models.Product
			err := skurows.Scan(&r.SKU, &r.Manufacturer, &r.ManufacturerPart, &r.Description, &r.ProcessRequest, &r.SortingRequest, &r.Unit, &r.UnitPrice, &r.Currency, &r.OrderQty, &r.Modified, &r.Reorder, &r.InventoryQty, &r.Season)
			if err != nil {
				return handler.Handleerror(pingErr), orders
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

func orderupdatesql(order int, tracking string, comment string, status string, user models.User) (message models.Message) {
	//Test Connection
	pingErr := config.DB.Ping()
	if pingErr != nil {
		config.DB, message = config.Opendb()
		return handler.Handleerror(pingErr)
	}

	//Build the Query
	log.WithFields(log.Fields{"username": user.Username}).Debug("Building Query...")
	newquery := "UPDATE `orders` SET `trackingnum`=?,`comments`=?,`status`=? WHERE ordernum = ?"

	//Run Query
	rows, err := config.DB.Query(newquery, tracking, comment, status, order)
	defer rows.Close()
	if err != nil {
		return handler.Handleerror(err)
	}
	message.Body = "Successfully updated order " + strconv.Itoa(order)
	message.Success = true
	//Logging
	log.WithFields(log.Fields{"username": user.Username}).Info("Updated Order ", strconv.Itoa(order))
	return message
}

// func userUpdateHandler(w http.ResponseWriter, r *http.Request) {

// 	//Test Connection
// 	pingErr := config.DB.Ping()
// 	if pingErr != nil {
// 		config.DB, _ = config.Opendb()
// 		// return handler.Handleerror(pingErr)
// 	}

// 	// Get the form values from the request
// 	username := r.FormValue("username")
// 	usercode := r.FormValue("usercode")
// 	role := r.FormValue("role")
// 	manager := r.FormValue("manager")
// 	var sorting int
// 	if r.FormValue("sorting") == "true" {
// 		sorting = 1
// 	} else {
// 		sorting = 0
// 	}
// 	// sorting, _ := strconv.ParseBool(r.FormValue("sorting"))
// 	println(username, usercode, role, sorting)

// 	// If usercode is empty, find the current max value and increment it
// 	if usercode == "" {
// 		var maxUsercode int
// 		err := config.DB.QueryRow("SELECT MAX(usercode) FROM orders.users").Scan(&maxUsercode)
// 		if err != nil {
// 			// Handle error
// 			handler.Handleerror(err)
// 		}
// 		usercode = strconv.Itoa(maxUsercode + 1)
// 	}

// 	// Prepare the SQL statement for inserting the data
// 	//Logging
// 	log.Info("Creating Query")
// 	newquery := "REPLACE INTO orders.users (username, usercode, permissions, sorting,manager) VALUES (?, ?, ?, ?, ?)"

// 	// Execute the SQL statement with the form values
// 	log.Info("Executing Query")
// 	rows, err := config.DB.Query(newquery, username, usercode, role, sorting, manager)
// 	defer rows.Close()

// 	if err != nil {
// 		// Handle error
// 		println(err)
// 	}

// 	// Redirect the user to the users page
// 	http.Redirect(w, r, "/users", http.StatusSeeOther)
// }

// func userDeleteHandler(w http.ResponseWriter, r *http.Request) {
// 	// Get the usercode value from the form
// 	usercode := r.FormValue("usercode")

// 	// Prepare the SQL statement for deleting the user
// 	stmt, err := config.DB.Prepare("UPDATE orders.users SET active = 0 WHERE usercode = ?")
// 	if err != nil {
// 		// Handle error
// 		println(err)
// 		http.Error(w, "Failed to update user information.", http.StatusInternalServerError)
// 		return
// 	}
// 	defer stmt.Close()

// 	// Execute the SQL statement with the usercode value
// 	_, err = stmt.Exec(usercode)
// 	if err != nil {
// 		// Handle error
// 		println(err)
// 		http.Error(w, "Failed to update user information.", http.StatusInternalServerError)
// 		return
// 	}

// 	// Redirect the user to the users page
// 	w.Write([]byte("User information updated successfully."))
// }

// func listorders(user User) (message Message, orders []Order) {
// 	//Debug
// 	log.WithFields(log.Fields{"username": user.Username}).Debug("Getting Orders...")

// 	//Test Connection
// 	pingErr := config.DB.Ping()
// 	if pingErr != nil {
// 		config.DB, message = config.Opendb()
// 		return handler.Handleerror(pingErr), orders
// 	}
// 	//Build the Query
// 	newquery := "SELECT ordernum,trackingnum,comments,manufacturer,status FROM `orders` WHERE 1"

// 	//Run Query
// 	rows, err := config.DB.Query(newquery)
// 	defer rows.Close()
// 	if err != nil {
// 		return handler.Handleerror(err), orders
// 	}

// 	//Pull Data
// 	for rows.Next() {
// 		var r Order
// 		err := rows.Scan(&r.Ordernum, &r.Tracking, &r.Comments, &r.Manufacturer, &r.Status)
// 		if err != nil {
// 			return handler.Handleerror(err), orders
// 		}
// 		orders = append(orders, r)
// 	}
// 	return message, orders
// }

func nextorder(manufacturer string, user models.User) (message models.Message, order models.Order) {
	// Get a database handle.
	// var err error
	var ordernum int
	//Test Connection
	pingErr := config.DB.Ping()
	if pingErr != nil {
		config.DB, message = config.Opendb()
		return handler.Handleerror(pingErr), order
	}
	//Build the Query
	newquery := "SELECT MAX(ordernum) ordernum  FROM `orders` WHERE 1"

	rows, err := config.DB.Query(newquery)
	defer rows.Close()
	if err != nil {
		return handler.Handleerror(err), order
	}
	// var val string
	if rows.Next() {
		rows.Scan(&ordernum)
	}
	ordernum += 1
	log.WithFields(log.Fields{"username": user.Username}).Debug(manufacturer, "-", ordernum)

	//insert new order into database
	newquery = "INSERT INTO orders (`ordernum`,`manufacturer`) VALUES (?,?)"
	orderinsert, err := config.DB.Query(newquery, ordernum, manufacturer)
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
func Reorderlist(user models.User) (message models.Message, orders []models.Order) {
	// Get a database handle.
	var err error

	// Test Connection
	pingErr := config.DB.Ping()
	if pingErr != nil {
		config.DB, message = config.Opendb()
		return handler.Handleerror(pingErr), orders
	}

	// Build the Query with pagination
	newquery := "SELECT skus.manufacturer_code, manufacturer_code.name FROM `skus` left join manufacturer_code on skus.manufacturer_code = manufacturer_code.code WHERE inventory_qty = 0 and reorder = 1 and manufacturer_code != '' group by skus.manufacturer_code, manufacturer_code.name"
	orderrows, err := config.DB.Query(newquery)
	if err != nil {
		return handler.Handleerror(pingErr), orders
	}
	defer orderrows.Close()
	//Pull Data
	for orderrows.Next() {
		var r models.Order
		err := orderrows.Scan(&r.Manufacturer, &r.ManufacturerName)
		if err != nil {
			return handler.Handleerror(pingErr), orders
		}
		// //Build the Query for the skus in the order
		// newquery := "SELECT a.sku_internal,`manufacturer_code`,`sku_manufacturer`,`product_option`,`processing_request`,`sorting_request`,`unit`,`unit_price`,`Currency`,`order_qty`,`modified`,`reorder`,`inventory_qty`,season,url_thumb,url_standard FROM `skus` a LEFT JOIN (select sku_internal FROM orderskus a left join orders b on a.ordernum = b.ordernum where status != 'Closed') b on a.sku_internal = b.sku_internal WHERE inventory_qty = 0 and reorder = 1 and b.sku_internal is null and manufacturer_code = ?"
		// skurows, err := config.DB.Query(newquery, r.Manufacturer)
		// if err != nil {
		// 	return handler.Handleerror(pingErr), orders
		// }
		// var skus []Product
		// defer skurows.Close()
		// for skurows.Next() {
		// 	var r Product
		// 	err := skurows.Scan(&r.SKU, &r.Manufacturer, &r.ManufacturerPart, &r.Description, &r.ProcessRequest, &r.SortingRequest, &r.Unit, &r.UnitPrice, &r.Currency, &r.OrderQty, &r.Modified, &r.Reorder, &r.InventoryQty, &r.Season, &r.Image.URL_Thumb, &r.Image.URL_Standard)
		// 	if err != nil {
		// 		return handler.Handleerror(pingErr), orders
		// 	}
		// 	skus = append(skus, r)
		// }
		// r.Products = skus
		//Append to the orders
		orders = append(orders, r)
	}

	return message, orders
}

func sortrequestdeletesql(requestid int, user models.User) (message models.Message) {
	//Debug
	log.WithFields(log.Fields{"username": user.Username}).Info("Deleting order ", "...")

	//Test Connection
	pingErr := config.DB.Ping()
	if pingErr != nil {
		config.DB, message = config.Opendb()
		return handler.Handleerror(pingErr)
	}

	//Build the Query
	newquery := "DELETE FROM `sortrequest` WHERE requestid = ?"
	rows, err := config.DB.Query(newquery, requestid)
	rows.Close()
	if err != nil {
		return handler.Handleerror(err)
	}

	message.Success = true
	message.Title = "Success"
	message.Body = "Successfully deleted Sorting Request ID  " + strconv.Itoa(requestid)
	return message
}
