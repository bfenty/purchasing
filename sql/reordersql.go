package sql

import (
	"purchasing/config"
	"purchasing/handler"
	"purchasing/models"
)

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
