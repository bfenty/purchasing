package models

import "time"

// this map stores the users sessions. For larger scale applications, you can use a database or cache for this purpose
var Sessions = map[string]Session{}

// each session contains the username of the user and the time at which it expires
type Session struct {
	Username string
	Expiry   time.Time
}

type Page struct {
	Title         string
	Date          string
	Layout        string
	Message       Message
	Permission    User
	ProductList   []Product
	Orders        []Order
	SortRequests  []SortRequest
	SortRequests2 []SortRequest
	Users         []User
}
