package models

type ProductBC struct {
	Data []struct {
		ID                    int           `json:"id"`
		Sku                   string        `json:"sku"`
		Brand_ID              int           `json:"brand_id"`
		InventoryLevel        int           `json:"inventory_level"`
		InventoryWarningLevel int           `json:"inventory_warning_level"`
		MPN                   string        `json:"mpn"`
		Detail                []Customfield `json:"custom_fields"`
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

type Product struct {
	SKU              string
	Description      *string
	Manufacturer     *string
	ManufacturerPart *string
	ProcessRequest   *string
	SortingRequest   *string
	Unit             *string
	UnitPrice        *float64
	Currency         *string
	OrderQty         *int
	Modified         *string
	Reorder          *bool
	InventoryQty     *int
	Season           *string
	Image            Image
}

type Image struct {
	URL_Standard *string `json:"url_standard"`
	URL_Thumb    *string `json:"url_thumbnail"`
	URL_Tiny     *string `json:"url_tiny"`
}

type Customfield struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Sku struct {
	SKU       string
	Qty       int
	ID        int
	Factory   int
	SupplySKU string
	Skuimage  Image
}

var Skulist []Sku
