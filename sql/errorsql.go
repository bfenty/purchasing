package sql

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"purchasing/api"
	"purchasing/config"
	"purchasing/handler"
	"sort"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

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
