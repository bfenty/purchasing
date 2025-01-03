package models

import "time"

// we'll use this method later to determine if the session has expired
func (s Session) isExpired() bool {
	return s.Expiry.Before(time.Now())
}

// Create a struct that models the structure of a user in the request body
type Credentials struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type User struct {
	Username    string
	Usercode    int
	Role        string
	Sorting     bool
	Manager     string
	Management  bool
	Permissions Permissions
}

type Permissions struct {
	Admin     bool
	Mgmt      bool
	Receiving bool
	Sorting   bool
}
