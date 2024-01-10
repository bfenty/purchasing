package main

import (
	// "encoding/csv"

	"encoding/json"
	"io/ioutil"

	// "log"
	"net/http"
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

type product struct {
	Data []struct {
		ID                    int           `json:"id"`
		Sku                   string        `json:"sku"`
		Brand_ID              int           `json:"brand_id"`
		InventoryLevel        int           `json:"inventory_level"`
		InventoryWarningLevel int           `json:"inventory_warning_level"`
		MPN                   string        `json:"mpn"`
		Detail                []customfield `json:"custom_fields"`
		Images                []Image       `json:"images"`
	} `json:"data"`
	Meta struct {
		Pagination struct {
			Total       int `json:"total"`
			Count       int `json:"count"`
			PerPage     int `json:"per_page"`
			CurrentPage int `json:"current_page"`
			TotalPages  int `json:"total_pages"`
			Links       struct {
				Next    string `json:"next"`
				Current string `json:"current"`
			} `json:"links"`
			TooMany bool `json:"too_many"`
		} `json:"pagination"`
	} `json:"meta"`
}

type Image struct {
	URL_Standard *string `json:"url_standard"`
	URL_Thumb    *string `json:"url_thumbnail"`
	URL_Tiny     *string `json:"url_tiny"`
}

type customfield struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type sku struct {
	SKU       string
	Qty       int
	ID        int
	Factory   int
	SupplySKU string
	Skuimage  Image
}

var skulist []sku

// Creates the URL by combining the url and link
func urlmake(url string, linkvalue string) (urlfinal string) {
	value := url + linkvalue
	log.Debug(value)
	return value
}

// jsonLoad retrieves product data from the specified URL and unmarshals it into a 'product' struct.
// It makes an HTTP GET request to the provided URL, reads the response, and parses the JSON data.
// The function logs various stages of execution for debugging purposes.
func jsonLoad(url string) (products product) {
	log.Debug("Loading JSON from URL: ", url)

	// Define the HTTP client with a timeout
	commerceClient := http.Client{
		Timeout: time.Second * 20, // Timeout after 20 seconds
	}

	// Create an HTTP GET request
	log.Debug("Creating HTTP GET request")
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "url": url}).Error("Failed to create HTTP request")
	}

	// Setup the request header
	req.Header.Set("User-Agent", "commerce-client")
	req.Header.Add("x-auth-token", os.Getenv("BIGCOMMERCE_TOKEN"))

	// Perform the HTTP request
	res, getErr := commerceClient.Do(req)
	if getErr != nil {
		log.WithFields(log.Fields{"error": getErr, "url": url}).Error("Failed to execute HTTP request")
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	// Read the response body
	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.WithFields(log.Fields{"error": readErr}).Error("Failed to read response body")
	}

	log.Debug("Response Body: ", string(body))

	// Unmarshal the JSON response into 'products'
	products = product{}
	jsonErr := json.Unmarshal(body, &products)
	if jsonErr != nil {
		log.WithFields(log.Fields{"error": jsonErr}).Error("Failed to unmarshal JSON")
	}
	log.Debug("Retrieved Products: ", products)
	return products
}

// printProducts processes a slice of products, extracting relevant data into a 'sku' struct.
// It also updates quantities based on the processed data. The function returns the current page
// number and the link to the next page of products.
func printProducts(products product) (page int, link string) {
	log.Debug("Processing products slice")

	for i, prod := range products.Data {
		tempsku := sku{
			SKU:       prod.Sku,
			Qty:       prod.InventoryLevel,
			ID:        prod.ID,
			Factory:   prod.Brand_ID,
			SupplySKU: prod.MPN,
		}

		if len(prod.Images) > 0 {
			tempsku.Skuimage = prod.Images[0]
		}

		skulist = append(skulist, tempsku)

		log.WithFields(log.Fields{
			"index":    i,
			"SKU":      tempsku.SKU,
			"Qty":      tempsku.Qty,
			"ImageURL": tempsku.Skuimage.URL_Standard,
		}).Debug("Processed product")
	}

	log.Debug("Updated SKU list: ", skulist)

	// Updating quantities for each SKU
	QTYUpdate(skulist)

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
	log.Debug("Fetching quantity for SKU: ", sku)

	storeid := os.Getenv("BIGCOMMERCE_STOREID")
	if storeid == "" {
		log.Fatal("BIGCOMMERCE_STOREID environment variable is not set")
	}
	limit := 250

	// Construct the initial request URL with query parameters
	link := "?sku:in=" + sku + "&include=images&include_fields=sku,inventory_level,inventory_warning_level,mpn,brand_id&limit=" + strconv.Itoa(limit)
	url := "https://api.bigcommerce.com/stores/" + storeid + "/v3/catalog/products" + link

	// Fetch the first page to get the total number of pages
	initialProducts := jsonLoad(urlmake(url, link))
	if len(initialProducts.Data) == 0 {
		log.WithFields(log.Fields{"SKU": sku}).Error("No records found for the given SKU")
		return // Exit the function as no data to process
	}
	totalpages := initialProducts.Meta.Pagination.TotalPages
	log.WithFields(log.Fields{"TotalPages": totalpages, "InitialURL": url}).Debug("Initial products fetch")

	// Loop through all pages
	for page := 1; page <= totalpages; page++ {
		currentPage, newLink := printProducts(jsonLoad(urlmake(url, link)))
		log.WithFields(log.Fields{"CurrentPage": currentPage, "NextPageLink": newLink}).Debug("Processed page")

		// Update the link for the next iteration
		link = newLink
	}

	log.Debug("Completed fetching and processing quantities for SKU: ", sku)
}

// Update QTY and IMG for products
func QTYUpdate(skus []sku) {

	for i := range skus {
		var newquery string = "UPDATE `skus` SET `inventory_qty`=?,url_thumb=?,url_standard=?,url_tiny=? WHERE sku_internal=REPLACE(?,' ','')"
		rows, err := db.Query(newquery, skus[i].Qty, skus[i].Skuimage.URL_Thumb, skus[i].Skuimage.URL_Standard, skus[i].Skuimage.URL_Tiny, skus[i].SKU)
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
