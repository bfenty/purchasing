package api

import (
	"fmt"
	"net/http"
	"purchasing/config"
	"purchasing/handler"
	"purchasing/models"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

type Field struct {
	Column string
	Alias  string
}

// ListCustomersAPI retrieves a list of customers with optional filters and pagination.
func ListCustomersAPI(w http.ResponseWriter, r *http.Request) {
	if err := config.DB.Ping(); err != nil {
		logrus.WithError(err).Error("Database operation failed")
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Database connection error"})
		return
	}

	filterConditions := extractQueryParams(r)
	var selectedFields = []Field{
		{"customer_email", "customer_email"},
		{"first_name", "first_name"},
		{"last_name", "last_name"},
		{"country", "country"},
		{"rebill_day", "rebill_day"},
		{"rebill_months", "rebill_months"},
		{"autorenew", "autorenew"},
		{"a.status", "cratejoy_status"},
		{"start_date", "start_date"},
		{"end_date", "end_date"},
		{"b.status", "mailchimp_status"},
	}
	table := "customers.cratejoy_subscriptions a LEFT JOIN customers.mailchimp b on a.customer_email = b.email"
	page, limit := getPaginationParams(r)
	offset := (page - 1) * limit

	queryArgs, queryBuilder, countQuery := buildQuery(table, selectedFields, filterConditions)

	var totalRecords int
	err := config.DB.QueryRow(countQuery, queryArgs...).Scan(&totalRecords)
	if err != nil {
		logrus.WithError(err).Error("Database operation failed")
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Error executing count query"})
		return
	}
	totalPages := (totalRecords + limit - 1) / limit

	customers, err := fetchCustomers(queryBuilder.String(), queryArgs, limit, offset)
	if err != nil {
		logrus.WithError(err).Error("Database operation failed")
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Error fetching customers"})
		return
	}

	handler.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"customers":    customers,
		"currentPage":  page,
		"perPage":      limit,
		"totalPages":   totalPages,
		"totalRecords": totalRecords,
	})
}

func extractQueryParams(r *http.Request) map[string]string {
	return map[string]string{
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
}

func getPaginationParams(r *http.Request) (int, int) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit < 1 {
		limit = 10
	}
	return page, limit
}

func buildQuery(table string, selectedFields []Field, filterConditions map[string]string) ([]interface{}, strings.Builder, string) {
	var queryArgs []interface{}
	var queryBuilder strings.Builder
	queryBuilder.WriteString("SELECT ")

	// Build SELECT fields using the structured slice
	fields := []string{}
	for _, field := range selectedFields {
		fields = append(fields, fmt.Sprintf("%s AS %s", field.Column, field.Alias))
	}

	queryBuilder.WriteString(strings.Join(fields, ", "))
	queryBuilder.WriteString(" FROM ")
	queryBuilder.WriteString(table)
	queryBuilder.WriteString(" WHERE 1")

	// Apply filter conditions correctly
	for column, value := range filterConditions {
		if value != "" {
			queryArgs = append(queryArgs, value)
			queryBuilder.WriteString(fmt.Sprintf(" AND %s = ?", column))
		}
	}

	// Build COUNT query for pagination
	countQuery := strings.Replace(queryBuilder.String(), "SELECT "+strings.Join(fields, ", "), "SELECT COUNT(*)", 1)

	// Log final query for debugging
	logrus.WithFields(logrus.Fields{"query": queryBuilder.String(), "args": queryArgs}).Debug("Final built SQL query")

	return queryArgs, queryBuilder, countQuery
}

func fetchCustomers(query string, args []interface{}, limit, offset int) ([]models.Customer, error) {
	queryArgs := append(args, limit, offset)
	query += " LIMIT ? OFFSET ?"
	logrus.Debug("Executing query: ", query, " with args: ", queryArgs)
	rows, err := config.DB.Query(query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var customers []models.Customer
	for rows.Next() {
		var customer models.Customer
		if err := rows.Scan(&customer.CustomerEmail, &customer.FirstName, &customer.LastName, &customer.Country, &customer.RebillDay, &customer.RebillMonths, &customer.AutoRenew, &customer.CratejoyStatus, &customer.StartDate, &customer.EndDate, &customer.MailchimpStatus); err != nil {
			return nil, err
		}
		customers = append(customers, customer)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return customers, nil
}
