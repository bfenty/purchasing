package api

import (
	"fmt"
	"net/http"
	"purchasing/config"
	"purchasing/handler"
	"purchasing/models"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

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
	pingErr := config.DB.Ping()
	if pingErr != nil {
		log.Debug(pingErr)
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Database connection error"})
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
	err := config.DB.QueryRow(countQuery, queryArgs...).Scan(&totalRecords)
	if err != nil {
		log.Debug(err)
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Error executing count query"})
		return
	}
	totalPages := (totalRecords + limit - 1) / limit // Calculate total pages

	// Add pagination to the main query
	queryArgs = append(queryArgs, limit, offset)
	query := queryBuilder.String() + " LIMIT ? OFFSET ?"
	log.WithFields(log.Fields{"query": query, "args": queryArgs}).Debug("Executing query")

	rows, err := config.DB.Query(query, queryArgs...)
	if err != nil {
		log.Debug(err)
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Error executing query"})
		return
	}
	defer rows.Close()

	var customers []models.Customer
	// Fetch rows
	for rows.Next() {
		var customer models.Customer
		err := rows.Scan(&customer.CustomerEmail, &customer.FirstName, &customer.LastName, &customer.Country, &customer.RebillDay, &customer.RebillMonths, &customer.AutoRenew, &customer.CratejoyStatus, &customer.StartDate, &customer.EndDate, &customer.MailchimpStatus)
		if err != nil {
			log.Error(err)
			handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Error scanning row"})
			return
		}
		customers = append(customers, customer)
	}

	// Handle any errors encountered during iteration
	if err = rows.Err(); err != nil {
		log.Debug(err)
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"message": "Error iterating over rows"})
		return
	}

	// Respond with the customer list and pagination info
	handler.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"customers":    customers,
		"currentPage":  page,
		"perPage":      limit,
		"totalPages":   totalPages,
		"totalRecords": totalRecords,
	})
}
