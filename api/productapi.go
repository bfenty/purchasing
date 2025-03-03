package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"purchasing/config"
	"purchasing/handler"
	"purchasing/models"
	"strconv"

	log "github.com/sirupsen/logrus"
)

// ProductList godoc
//	@Summary		List products
//	@Description	Retrieves a list of products, with optional filtering parameters
//	@Tags			products
//	@Accept			json
//	@Produce		json
//	@Security		X-API-Key
//	@Param			sku_internal		query		string					false	"Filter by internal SKU"
//	@Param			manufacturer_code	query		string					false	"Filter by manufacturer code"
//	@Param			sku_manufacturer	query		string					false	"Filter by manufacturer SKU"
//	@Param			processing_request	query		string					false	"Filter by processing request"
//	@Param			sorting_request		query		string					false	"Filter by sorting request"
//	@Param			unit				query		string					false	"Filter by unit"
//	@Param			unit_price			query		string					false	"Filter by unit price"
//	@Param			currency			query		string					false	"Filter by currency"
//	@Param			order_qty			query		string					false	"Filter by order quantity"
//	@Param			reorder				query		string					false	"Filter by reorder status"
//	@Param			season				query		string					false	"Filter by season"
//	@Param			page				query		int						false	"Page number (default: 1)"
//	@Param			limit				query		int						false	"Limit number of results (default: 100)"
//	@Success		200					{object}	map[string]interface{}	"List of products with pagination info"
//	@Failure		500					{object}	map[string]string		"Internal Server Error"
//	@Router			/api/products [get]

func ProductList(w http.ResponseWriter, r *http.Request) {
	log.Debug("Entering ProductListAPI")

	if err := config.DB.Ping(); err != nil {
		log.WithError(err).Error("Database connection error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	filterConditions := extractProductFilters(r)
	selectedFields := getProductFields()
	table := "purchasing.skus"
	page, limit := getPaginationParams(r)
	offset := (page - 1) * limit

	queryArgs, queryBuilder, countQuery := buildQuery(table, selectedFields, filterConditions)

	var totalRecords int
	if err := config.DB.QueryRow(countQuery, queryArgs...).Scan(&totalRecords); err != nil {
		log.WithError(err).Error("Error executing count query")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	totalPages := (totalRecords + limit - 1) / limit

	products, err := fetchProducts(queryBuilder.String(), queryArgs, limit, offset)
	if err != nil {
		log.WithError(err).Error("Error fetching products")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	responseData := map[string]interface{}{
		"currentPage":  page,
		"totalPages":   totalPages,
		"totalRecords": totalRecords,
		"perPage":      limit,
		"products":     products,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(responseData); err != nil {
		log.WithError(err).Error("Error encoding JSON")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	log.Debug("Exiting ProductListAPI")
}

func extractProductFilters(r *http.Request) map[string]string {
	return map[string]string{
		"sku_internal":       r.URL.Query().Get("sku"),
		"manufacturer_code":  r.URL.Query().Get("manufacturer"),
		"sku_manufacturer":   r.URL.Query().Get("manufacturerpart"),
		"processing_request": r.URL.Query().Get("processrequest"),
		"sorting_request":    r.URL.Query().Get("sortingrequest"),
		"unit":               r.URL.Query().Get("unit"),
		"unit_price":         r.URL.Query().Get("unitprice"),
		"currency":           r.URL.Query().Get("currency"),
		"order_qty":          r.URL.Query().Get("orderqty"),
		"season":             r.URL.Query().Get("season"),
	}
}

func getProductFields() []Field {
	return []Field{
		{"sku_internal", "sku_internal"},
		{"manufacturer_code", "manufacturer_code"},
		{"sku_manufacturer", "sku_manufacturer"},
		{"processing_request", "processing_request"},
		{"sorting_request", "sorting_request"},
		{"unit", "unit"},
		{"unit_price", "unit_price"},
		{"currency", "currency"},
		{"order_qty", "order_qty"},
		{"modified", "modified"},
		{"reorder", "reorder"},
		{"inventory_qty", "inventory_qty"},
		{"season", "season"},
		{"url_standard", "url_standard"},
		{"url_thumb", "url_thumb"},
		{"url_tiny", "url_tiny"},
		{"man_url", "man_url"},
	}
}

func fetchProducts(query string, args []interface{}, limit, offset int) ([]models.Product, error) {
	queryArgs := append(args, limit, offset)
	query += " LIMIT ? OFFSET ?"
	rows, err := config.DB.Query(query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(
			&p.SKU, &p.Manufacturer, &p.ManufacturerPart, &p.ProcessRequest,
			&p.SortingRequest, &p.Unit, &p.UnitPrice, &p.Currency, &p.OrderQty,
			&p.Modified, &p.Reorder, &p.InventoryQty, &p.Season,
			&p.Image.URL_Standard, &p.Image.URL_Thumb, &p.Image.URL_Tiny, &p.ManURL,
		); err != nil {
			return nil, err
		}
		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return products, nil
}

// InsertProduct godoc
//	@Summary		Insert new product
//	@Description	Adds a new product to the database
//	@Tags			products
//	@Accept			json
//	@Produce		json
//	@Security		X-API-Key
//	@Param			product	body		models.Product		true	"Product details to insert"
//	@Success		200		{object}	map[string]string	"Product successfully inserted"
//	@Failure		400		{object}	map[string]string	"Bad Request: Missing required fields"
//	@Failure		500		{object}	map[string]string	"Internal Server Error"
//	@Router			/api/productinsert [post]

func InsertProduct(w http.ResponseWriter, r *http.Request) {
	var p models.Product
	log.Debug("Starting InsertProduct")

	// Decode JSON request body
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		log.WithError(err).Error("Error decoding product data")
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal Server Error"})
		return
	}

	// Validate SKU
	if p.SKU == "" {
		log.Error("SKU is required for product insertion")
		handler.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Bad Request: SKU is required"})
		return
	}

	// Define table and selected fields
	table := "purchasing.skus"
	selectedFields := []Field{
		{"sku_internal", "SKU"},
		{"sku_manufacturer", "ManufacturerPart"},
		{"product_option", "Description"},
		{"manufacturer_code", "Manufacturer"},
		{"processing_request", "ProcessRequest"},
		{"unit", "Unit"},
		{"unit_price", "UnitPrice"},
		{"order_qty", "OrderQty"},
		{"reorder", "Reorder"},
		{"season", "Season"},
		{"inventory_qty", "InventoryQty"},
		{"Currency", "Currency"},
	}

	// Create filterConditions from product struct
	filterConditions := map[string]string{
		"sku_internal":       p.SKU,
		"sku_manufacturer":   derefString(p.ManufacturerPart),
		"product_option":     derefString(p.Description),
		"manufacturer_code":  derefString(p.Manufacturer),
		"processing_request": derefString(p.ProcessRequest),
		"unit":               derefString(p.Unit),
		"unit_price":         strconv.FormatFloat(derefFloat(p.UnitPrice), 'f', 2, 64),
		"order_qty":          strconv.Itoa(derefInt(p.OrderQty)),
		"reorder":            strconv.Itoa(boolToInt(derefBool(p.Reorder))),
		"season":             derefString(p.Season),
		"inventory_qty":      strconv.Itoa(derefInt(p.InventoryQty)),
		"Currency":           derefString(p.Currency),
	}

	// Build query using modified buildQuery function
	queryArgs, queryBuilder, _ := buildQuery(table, selectedFields, filterConditions, "REPLACE")

	log.Debug("Executing query: ", queryBuilder.String(), " with args: ", queryArgs)

	// Execute query
	_, err = config.DB.Exec(queryBuilder.String(), queryArgs...)
	if err != nil {
		log.WithError(err).Error("Error executing insert query")
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal Server Error"})
		return
	}

	log.WithFields(log.Fields{"SKU": p.SKU}).Info("Product successfully inserted")
	handler.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Product successfully inserted"})
}

// DeleteProduct godoc
//	@Summary		Delete a product
//	@Description	Deletes a product from the database based on its SKU
//	@Tags			products
//	@Accept			json
//	@Produce		json
//	@Security		X-API-Key
//	@Param			sku	body		string				true	"SKU of the product to delete"
//	@Success		200	{object}	map[string]string	"Product successfully deleted"
//	@Failure		400	{object}	map[string]string	"Bad Request: SKU is required"
//	@Failure		404	{object}	map[string]string	"Product not found"
//	@Failure		500	{object}	map[string]string	"Internal Server Error"
//	@Router			/api/productdelete [post]

func DeleteProduct(w http.ResponseWriter, r *http.Request) {
	log.Info("Received request to delete product")

	// Parse request body into a simple map
	var requestBody map[string]string
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		log.WithError(err).Error("Error decoding request body")
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to parse request"})
		return
	}

	// Extract and validate the SKU, which is required for product deletion
	sku, ok := requestBody["sku"]
	if !ok || sku == "" {
		log.Error("SKU is missing in delete request")
		handler.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "SKU is required"})
		return
	}

	log.WithFields(log.Fields{"SKU": sku}).Info("Parsed delete request")

	// Set up filter condition to match the product by SKU
	filterConditions := map[string]string{
		"sku_internal": sku,
	}

	// Use buildQuery to dynamically create the DELETE query
	queryArgs, queryBuilder, _ := buildQuery("purchasing.skus", nil, filterConditions, "DELETE")

	log.WithFields(log.Fields{
		"query": queryBuilder.String(),
		"args":  queryArgs,
	}).Debug("Executing delete query")

	// Execute the DELETE query
	result, err := config.DB.Exec(queryBuilder.String(), queryArgs...)
	if err != nil {
		log.WithFields(log.Fields{"SKU": sku, "error": err}).Error("Error executing delete query")
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete product"})
		return
	}

	// Check how many rows were affected (should be 1 if successful)
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.WithFields(log.Fields{"SKU": sku, "error": err}).Error("Error getting rows affected")
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to get deletion result"})
		return
	}

	// If no rows were affected, the product did not exist
	if rowsAffected == 0 {
		log.WithFields(log.Fields{"SKU": sku}).Warn("Product not found for deletion")
		handler.RespondWithJSON(w, http.StatusNotFound, map[string]string{"error": "Product not found"})
		return
	}

	// Successfully deleted the product
	log.WithFields(log.Fields{"SKU": sku, "rowsAffected": rowsAffected}).Info("Product deleted successfully")
	handler.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Product successfully deleted"})
}

// UpdateProduct godoc
//	@Summary		Update an existing product
//	@Description	Updates fields of a product in the database with the given SKU; only provided fields are updated
//	@Tags			products
//	@Accept			json
//	@Produce		json
//	@Security		X-API-Key
//	@Param			sku		body		string				true	"SKU of the product"
//	@Param			product	body		models.Product		true	"Fields to update in the product"
//	@Success		200		{object}	map[string]string	"Product successfully updated"
//	@Failure		400		{object}	map[string]string	"Bad Request: SKU is required"
//	@Failure		404		{object}	map[string]string	"Product not found"
//	@Failure		500		{object}	map[string]string	"Internal Server Error"
//	@Router			/api/productupdate [post]

func UpdateProduct(w http.ResponseWriter, r *http.Request) {
	log.Info("Received request to update product")

	// Decode request body into Product struct
	var p models.Product
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		log.WithError(err).Error("Error decoding product data")
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to parse request"})
		return
	}

	// Validate that SKU is provided, as it's required for identifying the product
	if p.SKU == "" {
		log.Error("SKU is required for product update")
		handler.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Bad Request: SKU is required"})
		return
	}

	// Create a map to hold the provided update values
	updateValues := map[string]interface{}{
		"sku_manufacturer":   p.ManufacturerPart,
		"product_option":     p.Description,
		"manufacturer_code":  p.Manufacturer,
		"processing_request": p.ProcessRequest,
		"sorting_request":    p.SortingRequest,
		"unit":               p.Unit,
		"unit_price":         p.UnitPrice,
		"order_qty":          p.OrderQty,
		"reorder":            p.Reorder,
		"season":             p.Season,
		"inventory_qty":      p.InventoryQty,
		"Currency":           p.Currency,
	}

	// Remove nil values from the updateValues map
	cleanedValues := make(map[string]string)
	for key, value := range updateValues {
		if value == nil {
			continue
		}

		switch v := value.(type) {
		case *string:
			if v != nil {
				cleanedValues[key] = *v
			}
		case *float64:
			if v != nil {
				cleanedValues[key] = fmt.Sprintf("%f", *v)
			}
		case *int:
			if v != nil {
				cleanedValues[key] = fmt.Sprintf("%d", *v)
			}
		case *bool:
			if v != nil {
				cleanedValues[key] = fmt.Sprintf("%t", *v)
			}
		default:
			log.WithFields(log.Fields{"field": key, "value": value}).Warn("Unexpected data type encountered")
		}
	}

	// Ensure there is at least one field to update
	if len(cleanedValues) == 0 {
		log.Warn("No fields provided for update")
		handler.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "No fields provided for update"})
		return
	}

	// Build the UPDATE query dynamically
	queryArgs, queryBuilder, _ := buildQuery("purchasing.skus", nil, cleanedValues, "UPDATE")
	queryArgs = append(queryArgs, p.SKU) // Add SKU as the WHERE clause condition

	log.WithFields(log.Fields{
		"query": queryBuilder.String(),
		"args":  queryArgs,
	}).Debug("Executing update query")

	// Execute the UPDATE query
	_, err := config.DB.Exec(queryBuilder.String(), queryArgs...)
	if err != nil {
		log.WithFields(log.Fields{"SKU": p.SKU, "error": err}).Error("Error executing update query")
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update product"})
		return
	}

	// Successfully updated the product
	log.WithFields(log.Fields{"SKU": p.SKU}).Info("Product successfully updated")
	handler.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Product successfully updated"})
}

// printProducts processes a slice of products, extracting relevant data into a 'sku' struct.
// It also updates quantities based on the processed data. The function returns the current page
// number and the link to the next page of products.
func printProducts(products models.ProductBC) (page int, link string) {
	log.Debug("Processing products slice")

	for i, prod := range products.Data {
		tempsku := models.Sku{
			SKU:       prod.Sku,
			Qty:       prod.InventoryLevel,
			ID:        prod.ID,
			Factory:   prod.Brand_ID,
			SupplySKU: prod.MPN,
		}

		if len(prod.Images) > 0 {
			tempsku.Skuimage = prod.Images[0]
		}

		models.Skulist = append(models.Skulist, tempsku)

		log.WithFields(log.Fields{
			"index":    i,
			"SKU":      tempsku.SKU,
			"Qty":      tempsku.Qty,
			"ImageURL": tempsku.Skuimage.URL_Standard,
		}).Debug("Processed product")
	}

	log.Debug("Updated SKU list: ", models.Skulist)

	// Updating quantities for each SKU
	QTYUpdate(models.Skulist)

	// Extracting pagination details
	link = products.Meta.Pagination.Links.Next
	page = products.Meta.Pagination.CurrentPage
	log.WithFields(log.Fields{
		"currentPage":  page,
		"nextPageLink": link,
	}).Debug("Pagination details")

	return page, link
}

// qty fetches product quantities for the given SKU(s) by making API requests to a specified endpoint.
// It iteratively fetches data for each page of products and processes the retrieved data.
func qty(sku string) {
	// Log the start of the process for the given SKU
	log.Debugf("Starting quantity fetch process for SKU: %s", sku)

	// Retrieve the store ID from the environment variable
	storeid := os.Getenv("BIGCOMMERCE_STOREID")
	if storeid == "" {
		log.Fatal("BIGCOMMERCE_STOREID environment variable is not set")
	}

	// Set the limit for the number of records per API call
	limit := 250
	baseURL := "https://api.bigcommerce.com/stores/" + storeid + "/v3/catalog/products"

	// Construct the initial part of the API URL
	link := "?sku:in=" + sku + "&include=images&include_fields=sku,inventory_level,inventory_warning_level,mpn,brand_id&limit=" + strconv.Itoa(limit)

	// Initialize the page counter
	page := 1

	// Loop through the pages of API results
	for {
		// Combine the base URL and additional parameters to form the full API URL
		fullURL := urlmake(baseURL, link)
		log.Debugf("Fetching data from URL: %s", fullURL)

		// Load the JSON data from the API
		products := handler.JsonLoad(fullURL)
		if len(products.Data) == 0 {
			// Log and break the loop if no records are found, especially on the first page
			if page == 1 {
				log.WithFields(log.Fields{"SKU": sku}).Error("No records found for the given SKU")
			} else {
				log.Debug("No more records to process")
			}
			break
		}

		// Print the products and get the link to the next page
		_, newLink := printProducts(products)
		log.WithFields(log.Fields{"CurrentPage": page, "NextPageLink": newLink}).Debug("Processed page")

		// Break the loop if there is no next page link or the last page has been reached
		if newLink == "" || page >= products.Meta.Pagination.TotalPages {
			log.Debug("Processed all pages")
			break
		}

		// Update the link for the next iteration and increment the page counter
		link = newLink
		page++
	}

	// Log the completion of the process for the given SKU
	log.Debugf("Completed processing for SKU: %s", sku)
}

// Update QTY and IMG for products
func QTYUpdate(skus []models.Sku) {

	for i := range skus {
		var newquery string = "UPDATE `skus` SET `inventory_qty`=?,url_thumb=?,url_standard=?,url_tiny=? WHERE sku_internal=REPLACE(?,' ','')"
		rows, err := config.DB.Query(newquery, skus[i].Qty, skus[i].Skuimage.URL_Thumb, skus[i].Skuimage.URL_Standard, skus[i].Skuimage.URL_Tiny, skus[i].SKU)
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

// Creates the URL by combining the url and link
func urlmake(url string, linkvalue string) (urlfinal string) {
	value := url + linkvalue
	log.Debug(value)
	return value
}
