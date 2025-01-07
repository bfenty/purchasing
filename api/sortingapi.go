package api

import (
	"encoding/json"
	"math"
	"purchasing/config"
	"purchasing/handler"
	"purchasing/models"
	"strconv"
	"strings"

	// "log"
	"fmt"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

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

	// Pagination parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 100 // Default limit
	}

	// Calculate offset
	offset := (page - 1) * limit

	//Gather Search Parameters
	queryParams := map[string]string{
		"requestid":        r.URL.Query().Get("requestid"),
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

	log.Debug("Parameters: ", queryParams)

	// Construct SQL query
	var queryArgs []interface{}
	var queryBuilder strings.Builder
	queryBuilder.WriteString("SELECT requestid, sku,description,instructions,weightin,weightout,pieces,hours,checkout,checkint,COALESCE(sorter,''),status,sku_manufacturer,prty from purchasing.sortrequest WHERE active=1 ")
	for param, value := range queryParams {
		if value == "NOT_NULL" {
			// Dynamically insert the parameter name into the query for NOT NULL conditions
			queryBuilder.WriteString(fmt.Sprintf(" AND %s IS NOT NULL", param))
			continue // Skip adding this to queryArgs since it's a special case
		}
		if value == "IS_NULL" {
			// Dynamically insert the parameter name into the query for NOT NULL conditions
			queryBuilder.WriteString(fmt.Sprintf(" AND %s IS NULL", param))
			continue // Skip adding this to queryArgs since it's a special case
		}
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
	err := config.DB.QueryRow(countQuery, queryArgs...).Scan(&totalRecords)
	if err != nil {
		log.Debug(err)
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Error executing count query"})
		return
	}
	totalPages := (totalRecords + limit - 1) / limit // Calculate total pages

	// Append limit and offset to the query
	queryArgs = append(queryArgs, limit, offset)
	query := queryBuilder.String() + " ORDER BY requestid DESC LIMIT ? OFFSET ? "

	// Execute query
	rows, err := config.DB.Query(query, queryArgs...)
	if err != nil {
		log.Debug(err)
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Error executing query"})
		return
	}
	defer rows.Close()

	// Process rows
	var sortRequests []models.SortRequest
	//Pull Data
	for rows.Next() {
		var r models.SortRequest
		err := rows.Scan(&r.ID, &r.SKU, &r.Description, &r.Instructions, &r.Weightin, &r.Weightout, &r.Pieces, &r.Hours, &r.Checkout, &r.Checkin, &r.Sorter, &r.Status, &r.ManufacturerPart, &r.Priority)
		if err != nil {
			log.Error(err)
			handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Error scanning row"})
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
			r.DifferencePercent = handler.FormatAsPercent(r.Difference / a) //Get the percentage of the weight in
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
	handler.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"sortRequests": sortRequests,
		"currentPage":  page,
		"totalPages":   totalPages,
		"totalRecords": totalRecords,
	})
}

// UpdateSortRequestAPI godoc
// @Summary Update a sort request
// @Description Update an existing sort request based on the provided request data
// @Tags sortrequests
// @Accept  json
// @Produce  json
// @Param request body models.SortRequest true "Sort Request Data"
// @Success 200 {object} map[string]string "Sort request updated successfully"
// @Failure 400 {object} map[string]string "Invalid request data"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /api/sorterrorupdate [post]
func UpdateSortRequestAPI(w http.ResponseWriter, r *http.Request) {
	var sortRequest models.SortRequest

	// Parse the request body
	err := json.NewDecoder(r.Body).Decode(&sortRequest)
	if err != nil {
		log.WithError(err).Error("Failed to parse request body")
		handler.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid request data"})
		return
	}

	log.WithFields(log.Fields{
		"requestid": sortRequest.ID,
		"sku":       sortRequest.SKU,
	}).Debug("Parsed sort request")

	// Validate required fields
	if sortRequest.ID == 0 {
		log.Error("Request ID is missing or invalid")
		handler.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"message": "Request ID is required"})
		return
	}

	// Dynamically build the update query based on non-nil fields
	var queryBuilder strings.Builder
	var args []interface{}
	queryBuilder.WriteString("UPDATE purchasing.sortrequest SET ")

	if sortRequest.SKU != "" {
		queryBuilder.WriteString("sku = ?, ")
		args = append(args, sortRequest.SKU)
	}
	if sortRequest.Description != nil {
		queryBuilder.WriteString("description = ?, ")
		args = append(args, *sortRequest.Description)
	}
	if sortRequest.ManufacturerPart != nil {
		queryBuilder.WriteString("sku_manufacturer = ?, ")
		args = append(args, *sortRequest.ManufacturerPart)
	}
	if sortRequest.Instructions != nil {
		queryBuilder.WriteString("instructions = ?, ")
		args = append(args, *sortRequest.Instructions)
	}
	if sortRequest.Weightin != nil {
		queryBuilder.WriteString("weightin = ?, ")
		args = append(args, *sortRequest.Weightin)
	}
	if sortRequest.Weightout != nil {
		queryBuilder.WriteString("weightout = ?, ")
		args = append(args, *sortRequest.Weightout)
	}
	if sortRequest.Pieces != nil {
		queryBuilder.WriteString("pieces = ?, ")
		args = append(args, *sortRequest.Pieces)
	}
	if sortRequest.Hours != nil {
		queryBuilder.WriteString("hours = ?, ")
		args = append(args, *sortRequest.Hours)
	}
	if sortRequest.Checkout != nil {
		queryBuilder.WriteString("checkout = ?, ")
		args = append(args, *sortRequest.Checkout)
	}
	if sortRequest.Checkin != nil {
		queryBuilder.WriteString("checkint = ?, ")
		args = append(args, *sortRequest.Checkin)
	}
	if sortRequest.Sorter != "" {
		queryBuilder.WriteString("sorter = ?, ")
		args = append(args, sortRequest.Sorter)
	}
	if sortRequest.Status != "" {
		queryBuilder.WriteString("status = ?, ")
		args = append(args, sortRequest.Status)
	}
	if sortRequest.Priority != 0 {
		queryBuilder.WriteString("prty = ?, ")
		args = append(args, sortRequest.Priority)
	}
	// Include the Active field
	if sortRequest.Active != nil {
		queryBuilder.WriteString("active = ?, ")
		args = append(args, *sortRequest.Active)
	}

	// Remove the trailing comma and space
	query := strings.TrimSuffix(queryBuilder.String(), ", ")
	query += " WHERE requestid = ?"
	args = append(args, sortRequest.ID)

	log.WithFields(log.Fields{
		"query": query,
		"args":  args,
	}).Debug("Executing update query")

	// Execute the query
	_, err = config.DB.Exec(query, args...)
	if err != nil {
		log.WithError(err).Error("Failed to update sort request")
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Error updating sort request"})
		return
	}

	// Respond with success
	log.WithFields(log.Fields{"requestID": sortRequest.ID}).Info("Sort request updated successfully")
	handler.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Sort request updated successfully"})
}

// func LookupRequestID(w http.ResponseWriter, r *http.Request) {
// 	requestID := r.URL.Query().Get("requestid")

// 	// Query the database for the sorter and description based on the requestid
// 	var sorter, description string
// 	err := config.DB.QueryRow("SELECT sorter, description FROM purchasing.sortrequest WHERE requestid = ?", requestID).Scan(&sorter, &description)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	// Return the sorter and description as a JSON object
// 	jsonObj := map[string]string{"sorter": sorter, "description": description}
// 	jsonBytes, err := json.Marshal(jsonObj)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	w.Write(jsonBytes)
// }

// // List of all sorting requests
// func listsortrequests(user User, action string, r *http.Request) (message Message, sortrequests []SortRequest) {
// 	//Debug
// 	log.WithFields(log.Fields{"username": user.Username}).Debug("Getting Sort Requests...")

// 	//Test Connection
// 	pingErr := db.Ping()
// 	if pingErr != nil {
// 		db, message = opendb()
// 		return handler.Handleerror(pingErr), sortrequests
// 	}

// 	//Gather Search Parameters
// 	queryParams := map[string]string{
// 		"sku":              r.URL.Query().Get("search-sku"),
// 		"description":      r.URL.Query().Get("search-description"),
// 		"sku_manufacturer": r.URL.Query().Get("search-manufacturerpart"),
// 		"instructions":     r.URL.Query().Get("search-instructions"),
// 		"weightout":        r.URL.Query().Get("search-weightout"),
// 		"weightin":         r.URL.Query().Get("search-weightin"),
// 		"pieces":           r.URL.Query().Get("search-pieces"),
// 		"hours":            r.URL.Query().Get("search-hours"),
// 		"checkout":         r.URL.Query().Get("search-checkout"),
// 		"checkint":         r.URL.Query().Get("search-checkin"),
// 		"sorter":           r.URL.Query().Get("search-sorter"),
// 		"status":           r.URL.Query().Get("search-status"),
// 		"prty":             r.URL.Query().Get("search-priority"),
// 	}

// 	var i []interface{}
// 	var newquery string

// 	//Build the Query
// 	if action == "all" {
// 		//retrieves all records
// 		newquery = "SELECT requestid, sku,description,instructions,weightin,weightout,pieces,hours,checkout,checkint,COALESCE(sorter,''),status,sku_manufacturer,prty from sortrequest WHERE active=1 "
// 		for param, value := range queryParams {
// 			if value != "" {
// 				value = value + "%"
// 				i = append(i, value)
// 				newquery += fmt.Sprintf(" AND %s LIKE ?", param)
// 			}
// 		}
// 		newquery += " order by 1 desc"
// 	} else if action == "checkout" {
// 		//retrieves only records that have not been checked out yet
// 		newquery = "SELECT requestid, sku,description,instructions,weightin,weightout,pieces,hours,checkout,checkint,COALESCE(sorter,''),status,sku_manufacturer,prty from sortrequest WHERE active=1 AND status = 'new'"
// 		// if permission.Perms == "sorting" {
// 		// 	newquery += " AND sorter = '" + user.Username + "'"
// 		// }
// 		newquery += " order by prty desc, 1"
// 	} else if action == "checkin" {
// 		//retrieves only records that have not been checked in yet
// 		newquery = "SELECT requestid, sku,description,instructions,weightin,weightout,pieces,hours,checkout,checkint,COALESCE(sorter,''),status,sku_manufacturer,prty from sortrequest WHERE active=1 AND status = 'checkout'"
// 		if user.Role == "sorting" {
// 			newquery += " AND sorter = '" + user.Username + "'"
// 		}
// 		newquery += " order by 1"
// 	} else if action == "receiving" {
// 		//retrieves only records that have been checked back in
// 		newquery = "SELECT requestid, sku,description,instructions,weightin,weightout,pieces,hours,checkout,checkint,COALESCE(sorter,''),status,sku_manufacturer,prty from sortrequest WHERE active=1 AND status = 'checkin' order by 1 desc"
// 	}

// 	newquery += " limit 200"

// 	//Run Query
// 	log.WithFields(log.Fields{"username": user.Username}).Debug(i...) //debug variables map
// 	log.WithFields(log.Fields{"username": user.Username}).Debug("Running Product List")
// 	log.WithFields(log.Fields{"username": user.Username}).Debug(newquery)
// 	log.WithFields(log.Fields{"username": user.Username}).Debug(user.Role)
// 	rows, err := db.Query(newquery, i...)
// 	defer rows.Close()
// 	if err != nil {
// 		return handler.Handleerror(err), sortrequests
// 	}

// 	//Pull Data
// 	for rows.Next() {
// 		var r SortRequest
// 		err := rows.Scan(&r.ID, &r.SKU, &r.Description, &r.Instructions, &r.Weightin, &r.Weightout, &r.Pieces, &r.Hours, &r.Checkout, &r.Checkin, &r.Sorter, &r.Status, &r.ManufacturerPart, &r.Priority)
// 		if err != nil {
// 			return handler.Handleerror(err), sortrequests
// 		}
// 		var a float64
// 		var b float64
// 		var c float64
// 		a = *r.Weightin
// 		b = *r.Weightout
// 		if r.Pieces != nil {
// 			c = float64(*r.Pieces)
// 		} else {
// 			// Handle the case where r.Pieces is nil
// 			c = 0.0
// 		}
// 		r.Difference = a - b - (c * 0.4555)               //0.4555 is the bag weight in grams
// 		r.Difference = math.Round(r.Difference*100) / 100 // Round to 2 decimal places
// 		if a != 0 {
// 			r.DifferencePercent = formatAsPercent(r.Difference / a) //Get the percentage of the weight in
// 		}
// 		if r.Difference < (-0.1*a) && a != 0 {
// 			r.Warn = true
// 		}
// 		r.Difference = -r.Difference
// 		// log.Info(c)
// 		sortrequests = append(sortrequests, r)
// 	}
// 	return message, sortrequests
// }

// // Sorting Insert
// func Sortinginsert(w http.ResponseWriter, r *http.Request) {
// 	// Test DB Connection
// 	pingErr := db.Ping()
// 	if pingErr != nil {
// 		db, _ = opendb()
// 		handler.Handleerror2(pingErr, w) // send error message to AJAX request
// 		return
// 	}

// 	user := auth(w, r)

// 	// Read the request data as JSON
// 	log.WithFields(log.Fields{"username": user.Username}).Debug("Decoding JSON")
// 	var data map[string]interface{}
// 	err := json.NewDecoder(r.Body).Decode(&data)
// 	if err != nil {
// 		handler.Handleerror2(err, w) // send error message to AJAX request
// 		return
// 	}

// 	log.WithFields(log.Fields{"username": user.Username, "request": data["requestid"]}).Debug("Fixing values")
// 	// Rename "sku_manufacturer" key to match the database column name
// 	if val, ok := data["manufacturerpart"]; ok {
// 		delete(data, "manufacturerpart")
// 		data["sku_manufacturer"] = val
// 	}

// 	// Rename "checkin" key to match the database column name
// 	if val, ok := data["checkin"]; ok {
// 		delete(data, "checkin")
// 		data["checkint"] = val
// 	}

// 	//fix <nil> request ID
// 	if data["requestid"] == "<nil>" {
// 		log.WithFields(log.Fields{"username": user.Username, "request": data["requestid"]}).Debug("requestid is nil")
// 		data["requestid"] = ""
// 	}

// 	// Remove the "difference" and "layout" fields, which are not in the database
// 	delete(data, "difference")
// 	delete(data, "layout")
// 	log.WithFields(log.Fields{"username": user.Username, "request": data["requestid"]}).Debug(data)

// 	// Check if the status is being updated to 'checkin'
// 	if data["status"] == "checkin" {
// 		// Check that hours are not blank
// 		if data["hours"] == nil || data["hours"] == "" {
// 			// Return an error message if hours are blank
// 			message := Message{Title: "Error", Body: "Hours cannot be blank when updating status to 'checkin'", Success: false}
// 			response, err := json.Marshal(message)
// 			if err != nil {
// 				http.Error(w, err.Error(), http.StatusInternalServerError)
// 				return
// 			}
// 			w.Header().Set("Content-Type", "application/json")
// 			w.Write(response)
// 			return
// 		}
// 	}

// 	// Define variables
// 	var newquery string
// 	var values []interface{}
// 	var message Message

// 	// Construct SQL query based on request data
// 	log.WithFields(log.Fields{"username": user.Username, "request": data["requestid"]}).Debug("Constructing SQL")
// 	if data["requestid"] == nil || data["requestid"] == "" || data["requestid"] == "<nil>" { //if this is a new request
// 		newquery = "REPLACE INTO sortrequest ("
// 		for key, value := range data {
// 			if value == "<nil>" {
// 				value = "" // fix nil values being inserted
// 			}
// 			if value != nil && value != "" {
// 				newquery += "`" + key + "`,"
// 				values = append(values, value)
// 			}
// 			log.WithFields(log.Fields{"username": user.Username, "request": data["requestid"], key: value}).Debug("added to query")
// 		}
// 		newquery = newquery[:len(newquery)-1] + ") VALUES ("
// 		for range values {
// 			newquery += "?,"
// 		}
// 		newquery = newquery[:len(newquery)-1] + ")"
// 		// create success message and send it to AJAX request
// 		message = Message{Title: "Success", Body: "Successfully inserted request", Success: true}
// 	} else { //if updating an existing request
// 		newquery = "UPDATE sortrequest SET "
// 		for key, value := range data {
// 			if value == nil {
// 				value = "" // fix nil values being inserted
// 			}
// 			if value == "<nil>" {
// 				value = "" // fix nil values being inserted
// 			}
// 			if value != "" {
// 				newquery += "`" + key + "`=?,"
// 				values = append(values, value)
// 			}
// 		}
// 		newquery = newquery[:len(newquery)-1] //get rid of the last comma
// 		newquery += " WHERE requestid=?"
// 		values = append(values, data["requestid"])
// 		// create success message and send it to AJAX request
// 		message = Message{Title: "Success", Body: "Successfully updated request", Success: true}
// 	}

// 	log.WithFields(log.Fields{
// 		"requestid": data["requestid"],
// 		"query":     newquery,
// 		"values":    values,
// 	}).Info("Sortinginsert: received data")

// 	stmt, err := db.Prepare(newquery)
// 	if err != nil {
// 		handler.Handleerror2(err, w) // send error message to AJAX request
// 		return
// 	}
// 	defer stmt.Close()

// 	_, err = stmt.Exec(values...)
// 	if err != nil {
// 		handler.Handleerror2(err, w) // send error message to AJAX request
// 		return
// 	}

// 	// Logging
// 	log.WithFields(log.Fields{
// 		"requestid": data["requestid"],
// 	}).Info("Sortinginsert: request processed")

// 	// encode message as JSON
// 	response, err := json.Marshal(message)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	// set content type to JSON and send response
// 	w.Header().Set("Content-Type", "application/json")
// 	w.Write(response)
// }
