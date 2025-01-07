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
	"strings"

	log "github.com/sirupsen/logrus"
)

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
	pingErr := config.DB.Ping()
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
		"season":             r.URL.Query().Get("season"),
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

	queryBuilder.WriteString("SELECT `sku_internal`,`manufacturer_code`,`sku_manufacturer`,`product_option`,`processing_request`,`sorting_request`,`unit`,`unit_price`,`Currency`,`order_qty`,`modified`,`reorder`,`inventory_qty`,season,url_standard,url_thumb,url_tiny,`man_url` FROM purchasing.skus WHERE 1")

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

	rows, err := config.DB.Query(queryBuilder.String(), queryArgs...)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error executing query")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.SKU, &p.Manufacturer, &p.ManufacturerPart, &p.Description, &p.ProcessRequest, &p.SortingRequest, &p.Unit, &p.UnitPrice, &p.Currency, &p.OrderQty, &p.Modified, &p.Reorder, &p.InventoryQty, &p.Season, &p.Image.URL_Standard, &p.Image.URL_Thumb, &p.Image.URL_Tiny, &p.ManURL); err != nil {
			log.WithFields(log.Fields{"error": err}).Error("Error scanning row")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		products = append(products, p)
	}

	// Calculate total number of records and total pages
	var totalRecords int
	err = config.DB.QueryRow("SELECT COUNT(*) FROM purchasing.skus").Scan(&totalRecords)
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
	var p models.Product
	// Log the start of the function
	log.Debug("Starting InsertProduct")

	// Attempt to decode the incoming request body into the Product struct
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error decoding product data")
		errMsg := fmt.Sprintf("Internal Server Error: %v", err)
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": errMsg})
		return
	}

	// Log the product being inserted for traceability
	log.WithFields(log.Fields{
		"SKU":              p.SKU,
		"Manufacturer":     p.Manufacturer,
		"ManufacturerPart": p.ManufacturerPart,
		"Description":      p.Description,
		"Currency":         p.Currency,
	}).Debug("Attempting to insert product")

	// Validate that SKU is provided
	if p.SKU == "" {
		log.Error("SKU is required for product insertion")
		handler.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Bad Request: SKU is required"})
		return
	}

	// SQL INSERT statement
	query := `REPLACE INTO purchasing.skus (sku_internal, sku_manufacturer, product_option, manufacturer_code, processing_request, unit, unit_price, order_qty, reorder, season, inventory_qty, Currency) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = config.DB.Exec(query, p.SKU, p.ManufacturerPart, p.Description, p.Manufacturer, p.ProcessRequest, p.Unit, p.UnitPrice, p.OrderQty, p.Reorder, p.Season, p.InventoryQty, p.Currency)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "SKU": p.SKU}).Error("Error executing insert query")
		errMsg := fmt.Sprintf("Internal Server Error: %v", err)
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": errMsg})
		return
	}

	// Log the successful insertion
	log.WithFields(log.Fields{"SKU": p.SKU}).Info("Product successfully inserted")

	// Optionally log the call to update quantity and image URLs
	log.Debug("Updating quantity and image URLs for SKU: ", p.SKU)
	qty(p.SKU)

	// Respond with success message
	handler.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Product successfully inserted"})
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
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": errMsg})
		return
	}

	sku, ok := requestBody["sku"]
	if !ok || sku == "" {
		log.Error("SKU is missing in delete request")
		handler.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "SKU is required"})
		return
	}

	log.WithFields(log.Fields{"SKU": sku}).Info("Parsed delete request")

	// SQL DELETE statement
	query := `DELETE FROM purchasing.skus WHERE sku_internal = ?`
	result, err := config.DB.Exec(query, sku)
	if err != nil {
		log.WithFields(log.Fields{"SKU": sku, "error": err}).Error("Error executing delete query")
		errMsg := fmt.Sprintf("Internal Server Error: %v", err)
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": errMsg})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.WithFields(log.Fields{"SKU": sku, "error": err}).Error("Error getting rows affected")
		errMsg := fmt.Sprintf("Internal Server Error: %v", err)
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": errMsg})
		return
	}

	if rowsAffected == 0 {
		log.WithFields(log.Fields{"SKU": sku}).Warn("Product not found for deletion")
		handler.RespondWithJSON(w, http.StatusNotFound, map[string]string{"error": "Product not found"})
		return
	}

	log.WithFields(log.Fields{"SKU": sku, "rowsAffected": rowsAffected}).Info("Product deleted successfully")
	handler.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Product successfully deleted"})
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
	var p models.Product
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error decoding product data")
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Internal Server Error: %v", err)})
		return
	}

	if p.SKU == "" {
		log.Error("SKU is required for product update")
		handler.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Bad Request: SKU is required"})
		return
	}

	var queryArgs []interface{}
	var queryBuilder strings.Builder
	queryBuilder.WriteString("UPDATE purchasing.skus SET ")

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

	_, err = config.DB.Exec(query, queryArgs...)
	if err != nil {
		log.WithFields(log.Fields{"SKU": p.SKU, "error": err}).Error("Error executing update query")
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Internal Server Error: %v", err)})
		return
	}

	//return JSON response
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
