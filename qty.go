package main

import (
	// "encoding/csv"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
	// log "github.com/sirupsen/logrus"
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
	fmt.Println(value)
	return value
}

// loads JSON and returns a slice
func jsonLoad(url string) (products product) {
	fmt.Println("Loading JSON")
	//Define the Request Client
	commerceClient := http.Client{
		Timeout: time.Second * 20, // Timeout after 2 seconds
	}

	//HTTP Request
	fmt.Println("HTTP Request formatting")
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	//Setup Header
	req.Header.Set("User-Agent", "commerce-client")
	req.Header.Add("x-auth-token", os.Getenv("BIGCOMMERCE_TOKEN"))

	res, getErr := commerceClient.Do(req)
	if getErr != nil {
		log.Fatal(getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	fmt.Println(string(body))

	//unmarshall JSON
	products = product{}
	jsonErr := json.Unmarshal(body, &products)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}
	fmt.Println("Products:", products)
	return products
}

// prints out the products slice
func printProducts(products product) (page int, link string) {
	var tempsku sku
	for i := range products.Data {
		tempsku.SKU = products.Data[i].Sku
		tempsku.Qty = products.Data[i].InventoryLevel
		tempsku.ID = products.Data[i].ID
		tempsku.Factory = products.Data[i].Brand_ID
		tempsku.SupplySKU = products.Data[i].MPN
		if len(products.Data[i].Images) > 0 {
			tempsku.Skuimage = products.Data[i].Images[0]
		}
		skulist = append(skulist, tempsku)
		//			}
	}
	fmt.Println("SKU LIST: ", skulist)
	QTYUpdate(skulist)
	link = products.Meta.Pagination.Links.Next
	return products.Meta.Pagination.CurrentPage, link
}

func qty(sku string) {
	//Define URL strings
	var url string
	var link string
	storeid := os.Getenv("BIGCOMMERCE_STOREID")
	limit := 250

	//Define the Request URL
	// sku := "CAP-263"
	link = "?sku=" + sku + "&include=images&include_fields=sku,inventory_level,inventory_warning_level,mpn,brand_id&limit=" + strconv.Itoa(limit)
	url = "https://api.bigcommerce.com/stores/" + storeid + "/v3/catalog/products"

	//Loop through the pages
	totalpages := jsonLoad(urlmake(url, link)).Meta.Pagination.TotalPages
	fmt.Println("Total Pages:", totalpages)
	i := 0
	for i < totalpages {
		page, newlink := printProducts(jsonLoad(urlmake(url, link)))
		fmt.Println("Next Page Query:", page, newlink)
		link = newlink
		i = page
	}
}
