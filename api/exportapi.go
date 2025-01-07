package api

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"purchasing/config"
	"strings"

	log "github.com/sirupsen/logrus"
)

// ExportDataAPI exports data as CSV based on the dataset and filters provided.
// @Summary Export data to CSV
// @Description Export data from the database as a CSV file.
// @Tags export
// @Accept json
// @Produce text/csv
// @Param dataset query string true "Dataset to export (e.g., 'products', 'sort_requests')"
// @Param filters query string false "Filters for the dataset (e.g., 'sku=ABC123')"
// @Param columns query string false "Comma-separated list of columns to include"
// @Success 200 {file} text/csv "CSV file with the exported data"
// @Failure 400 {object} map[string]string "Invalid request parameters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/export [get]
func ExportDataAPI(w http.ResponseWriter, r *http.Request) {
	dataset := r.URL.Query().Get("dataset")
	if dataset == "" {
		http.Error(w, "Dataset is required", http.StatusBadRequest)
		return
	}

	filters := r.URL.Query().Get("filters")
	columns := r.URL.Query().Get("columns")

	// Build the query dynamically based on dataset and filters
	var query string
	var args []interface{}

	switch dataset {
	case "products":
		query = "SELECT sku_internal, manufacturer_code, sku_manufacturer, product_option FROM purchasing.skus"
		if filters != "" {
			query += " WHERE " + strings.ReplaceAll(filters, "=", " LIKE ?")
			args = append(args, "%"+strings.Split(filters, "=")[1]+"%")
		}
	case "sort_requests":
		query = "SELECT requestid, sku, description, status FROM purchasing.sortrequest"
		if filters != "" {
			query += " WHERE " + strings.ReplaceAll(filters, "=", " LIKE ?")
			args = append(args, "%"+strings.Split(filters, "=")[1]+"%")
		}
	default:
		http.Error(w, "Unsupported dataset", http.StatusBadRequest)
		return
	}

	// Select specific columns if provided
	if columns != "" {
		cols := strings.Split(columns, ",")
		query = strings.Replace(query, "*", strings.Join(cols, ", "), 1)
	}

	// Execute the query
	rows, err := config.DB.Query(query, args...)
	if err != nil {
		log.WithError(err).Error("Error executing query")
		http.Error(w, "Error executing query", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Prepare the response header for CSV
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=%s.csv", dataset))

	// Write CSV data
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write headers
	columns, err := rows.Columns()
	if err != nil {
		log.WithError(err).Error("Error getting columns")
		http.Error(w, "Error fetching data", http.StatusInternalServerError)
		return
	}
	writer.Write(columns)

	// Write rows
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			log.WithError(err).Error("Error scanning row")
			http.Error(w, "Error fetching data", http.StatusInternalServerError)
			return
		}

		record := make([]string, len(columns))
		for i, value := range values {
			if value != nil {
				record[i] = fmt.Sprintf("%v", value)
			}
		}
		writer.Write(record)
	}

	if rows.Err() != nil {
		log.WithError(rows.Err()).Error("Error iterating over rows")
		http.Error(w, "Error fetching data", http.StatusInternalServerError)
		return
	}
}