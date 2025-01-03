package models

import "database/sql"

type Customer struct {
	CustomerEmail   string         `json:"customer_email"`
	FirstName       sql.NullString `json:"first_name"`
	LastName        sql.NullString `json:"last_name"`
	Country         sql.NullString `json:"country"`
	RebillDay       sql.NullInt32  `json:"rebill_day"`
	RebillMonths    sql.NullInt32  `json:"rebill_months"`
	AutoRenew       bool           `json:"autorenew"`
	CratejoyStatus  sql.NullString `json:"cratejoy_status"`
	StartDate       sql.NullString `json:"start_date"` // Assuming dates are in string format, change to time.Time if using date objects
	EndDate         sql.NullString `json:"end_date"`
	MailchimpStatus sql.NullString `json:"mailchimp_status"`
}
