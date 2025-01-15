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
	columns := r.URL.Query().Get("columns") // Query parameter for user-specified columns

	// Build the query dynamically
	var queryBuilder strings.Builder
	var args []interface{}

	switch dataset {
	case "products":
		queryBuilder.WriteString("SELECT sku_internal, manufacturer_code, sku_manufacturer, product_option FROM purchasing.skus")
	case "sort_requests":
		queryBuilder.WriteString("SELECT requestid, sku, description, status FROM purchasing.sortrequest")
	default:
		http.Error(w, "Unsupported dataset", http.StatusBadRequest)
		return
	}

	// Parse and add filters dynamically
	if filters != "" {
		filterParts := strings.Split(filters, "&") // Split the filters into key-value pairs
		if len(filterParts) > 0 {
			queryBuilder.WriteString(" WHERE ")
			for i, filter := range filterParts {
				keyValue := strings.Split(filter, "=")
				if len(keyValue) == 2 {
					if i > 0 {
						queryBuilder.WriteString(" AND ")
					}
					queryBuilder.WriteString(fmt.Sprintf("%s LIKE ?", keyValue[0]))
					args = append(args, "%"+keyValue[1]+"%")
				}
			}
		}
	}

	// Store the base query before modifying it for custom columns
	baseQuery := queryBuilder.String()

	// Add user-specified columns if provided
	if columns != "" {
		selectedColumns := strings.Split(columns, ",")
		columnList := strings.Join(selectedColumns, ", ")
		queryBuilder.Reset() // Clear the existing query
		queryBuilder.WriteString(fmt.Sprintf("SELECT %s FROM (%s) AS subquery", columnList, baseQuery))
	} else {
		queryBuilder.WriteString(baseQuery)
	}

	query := queryBuilder.String()

	log.WithFields(log.Fields{"query": query, "args": args}).Debug("Executing query")

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

	// Retrieve column names from the result set
	tableColumns, err := rows.Columns()
	if err != nil {
		log.WithError(err).Error("Error getting column names")
		http.Error(w, "Error fetching data", http.StatusInternalServerError)
		return
	}

	// Write column headers to the CSV
	writer.Write(tableColumns)

	// Write rows to the CSV
	for rows.Next() {
		values := make([]interface{}, len(tableColumns))
		valuePtrs := make([]interface{}, len(tableColumns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			log.WithError(err).Error("Error scanning row")
			http.Error(w, "Error fetching data", http.StatusInternalServerError)
			return
		}

		record := make([]string, len(tableColumns))
		for i, value := range values {
			switch v := value.(type) {
			case []byte:
				record[i] = string(v) // Convert byte slice to string
			case nil:
				record[i] = "" // Handle NULL values
			default:
				record[i] = fmt.Sprintf("%v", v) // Default string conversion
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
