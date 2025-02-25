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
//	@Summary		List sort requests
//	@Description	Retrieves a list of sort requests with optional filtering parameters
//	@Tags			sortrequests
//	@Accept			json
//	@Produce		json
//	@Param			requestid			query		string					false	"Filter by Request ID"
//	@Param			sku					query		string					false	"Filter by SKU"
//	@Param			description			query		string					false	"Filter by Description"
//	@Param			manufacturer_part	query		string					false	"Filter by Manufacturer Part"
//	@Param			instructions		query		string					false	"Filter by Instructions"
//	@Param			weightout			query		string					false	"Filter by Weight Out"
//	@Param			weightin			query		string					false	"Filter by Weight In"
//	@Param			pieces				query		string					false	"Filter by Pieces"
//	@Param			hours				query		string					false	"Filter by Hours"
//	@Param			checkout			query		string					false	"Filter by Checkout"
//	@Param			checkin				query		string					false	"Filter by Checkin"
//	@Param			sorter				query		string					false	"Filter by Sorter"
//	@Param			status				query		string					false	"Filter by Status"
//	@Param			priority			query		string					false	"Filter by Priority"
//	@Param			page				query		int						false	"Page number (default: 1)"
//	@Param			limit				query		int						false	"Number of records per page (default: 100)"
//	@Success		200					{object}	map[string]interface{}	"List of sort requests with pagination"
//	@Failure		500					{object}	map[string]string		"Internal Server Error"
//	@Router			/api/sortrequests [get]

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
//	@Summary		Update a sort request
//	@Description	Updates an existing sort request with the provided data
//	@Tags			sortrequests
//	@Accept			json
//	@Produce		json
//	@Param			request	body		models.SortRequest	true	"Sort request data to update"
//	@Success		200		{object}	map[string]string	"Sort request updated successfully"
//	@Failure		400		{object}	map[string]string	"Bad Request: Invalid input or missing request ID"
//	@Failure		500		{object}	map[string]string	"Internal Server Error"
//	@Router			/api/sorterrorupdate [post]

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
