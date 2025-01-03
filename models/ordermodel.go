package models

type Order struct {
	Ordernum         int
	Manufacturer     *string
	ManufacturerName *string
	Status           string
	Comments         *string
	Tracking         *string
	Products         []Product
}
