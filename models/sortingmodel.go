package models

type SortRequest struct {
	ID                int
	SKU               string
	Description       *string
	Instructions      *string
	Weightin          *float64
	Weightout         *float64
	Difference        float64
	DifferencePercent string
	Pieces            *int
	Hours             *float64
	Checkout          *string
	Checkin           *string
	Sorter            string
	Status            string
	ManufacturerPart  *string
	Priority          int
	Warn              bool
}
