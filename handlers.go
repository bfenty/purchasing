package main

import (
	// "encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// this map stores the users sessions. For larger scale applications, you can use a database or cache for this purpose
var sessions = map[string]session{}

// each session contains the username of the user and the time at which it expires
type session struct {
	username string
	expiry   time.Time
}

// we'll use this method later to determine if the session has expired
func (s session) isExpired() bool {
	return s.expiry.Before(time.Now())
}

// Create a struct that models the structure of a user in the request body
type Credentials struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

func Usercreate(w http.ResponseWriter, r *http.Request) {
	var creds Credentials
	var message Message
	var success bool
	// log.Debug("method:", r.Method) //get request method
	r.ParseForm()
	// logic part of log in
	creds.Username = r.FormValue("username")
	creds.Password = r.FormValue("password")
	if creds.Password != r.FormValue("password2") {
		message.Title = "Non-matching passwords"
		message.Body = "Passwords do not match"
		http.Redirect(w, r, "/signup?messagetitle="+message.Title+"&messagebody="+message.Body, http.StatusSeeOther)
		return
	}
	log.Debug("Creating user ", creds.Username, "...")
	message, success = Updatepass(creds.Username, creds.Password, r.FormValue("secret"))
	if success {
		http.Redirect(w, r, "/products?messagetitle="+message.Title+"&messagebody="+message.Body, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/signup?messagetitle="+message.Title+"&messagebody="+message.Body, http.StatusSeeOther)
	return
}

func Signin(w http.ResponseWriter, r *http.Request) {
	log.Debug("Logging in...")
	var creds Credentials
	// log.Debug("method:", r.Method) //get request method
	r.ParseForm()
	// logic part of log in
	creds.Username = r.FormValue("username")
	creds.Password = r.FormValue("password")
	user, message := userauth(creds.Username, creds.Password)
	log.Debug(message.Body)

	// If a password exists for the given user
	// AND, if it is the same as the password we received, the we can move ahead
	// if NOT, then we return an "Unauthorized" status
	if user.Role == "notfound" {
		log.Debug(message.Body)
		w.WriteHeader(http.StatusUnauthorized)
		// fmt.Fprintf(w, "Invalid username or password")
		fmt.Fprintf(w, message.Body)
		return
	}

	//redirect new user to the signup page
	if user.Role == "newuser" {
		log.Debug(message.Body)
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, "/signup")
		return
	}

	// Create a new random session token
	// we use the "github.com/google/uuid" library to generate UUIDs
	sessionToken := uuid.NewString()
	expiresAt := time.Now().Add(1800 * time.Second)
	// log.Debug("Authorized")

	// Set the token in the session map, along with the session information
	sessions[sessionToken] = session{
		username: creds.Username,
		expiry:   expiresAt,
	}

	// Replace the session map assignment with a database insert
	_, err := db.Exec("INSERT INTO purchasing.sessions (token, username, expiry) VALUES (?, ?, ?)", sessionToken, creds.Username, expiresAt)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error saving session to database")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Finally, we set the client cookie for "session_token" as the session token we just generated
	// we also set an expiry time of 120 seconds
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   sessionToken,
		Expires: expiresAt,
	})
	log.Debug(sessions)

	// Return success response with appropriate redirect URL based on user's role
	switch user.Role {
	case "sorting":
		log.Debug(message.Body)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "/checkout")
		return
	case "receiving":
		log.Debug(message.Body)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "/products")
		return
	case "newuser":
		log.Debug(message.Body)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "/signup")
		return
	default:
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "/productsinsert")
		return
	}
}

func auth(w http.ResponseWriter, r *http.Request) (user User) {
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			//Redirect Login
			http.Redirect(w, r, "/login?messagetitle=User Unauthorized&messagebody=Please login to use this site", http.StatusSeeOther)
			// If the cookie is not set, return an unauthorized status
			user.Role = "Unauthorized"
			return user
		}
	}
	sessionToken := c.Value

	// We then get the name of the user from our session map, where we set the session token
	// userSession, exists := sessions[sessionToken]
	// if !exists {
	// 	// If the session token is not present in session map, return an unauthorized error
	// 	// w.WriteHeader(http.StatusUnauthorized)
	// 	log.Debug("Unauthorized")
	// 	http.Redirect(w, r, "/login?messagetitle=User Unauthorized&messagebody=Please login to use this site", http.StatusSeeOther)
	// 	user.Role = "Unauthorized"
	// 	return user
	// }
	// if userSession.isExpired() {
	// 	delete(sessions, sessionToken)
	// 	// w.WriteHeader(http.StatusUnauthorized)
	// 	log.Debug("Unauthorized")
	// 	http.Redirect(w, r, "/login?messagetitle=User Unauthorized&messagebody=Please login to use this site", http.StatusSeeOther)

	// 	user.Role = "Unauthorized"
	// 	return user
	// }

	var username string
	var expiry time.Time
	err = db.QueryRow("SELECT username, expiry FROM sessions WHERE token = ?", sessionToken).Scan(&username, &expiry)
	if err != nil {
		// Handle no rows found or other errors
		log.Debug("Session token not found in database, redirecting to login")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return User{Role: "Unauthorized"}
	}

	if expiry.Before(time.Now()) {
		// Session is expired, delete it and redirect to login
		_, delErr := db.Exec("DELETE FROM sessions WHERE token = ?", sessionToken)
		if delErr != nil {
			log.WithFields(log.Fields{"error": delErr}).Error("Error deleting expired session")
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return User{Role: "Unauthorized"}
	}

	// Finally, return the welcome message to the user
	log.Debug("Authorized")
	// If the previous session is valid, create a new session token for the current user
	newSessionToken := uuid.NewString()
	expiresAt := time.Now().Add(1800 * time.Second)

	// First, delete the old session ID
	_, delErr := db.Exec("DELETE FROM sessions WHERE token = ?", sessionToken)
	if delErr != nil {
		log.WithFields(log.Fields{"error": delErr}).Error("Error deleting expired session")
	}

	// Replace the session map assignment with a database insert
	_, err = db.Exec("INSERT INTO sessions (token, username, expiry) VALUES (?, ?, ?)", newSessionToken, username, expiresAt)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error saving session to database")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Set the token in the session map, along with the user whom it represents
	// sessions[newSessionToken] = session{
	// 	username: userSession.username,
	// 	expiry:   expiresAt,
	// }

	// Delete the older session token
	delete(sessions, sessionToken)

	// Set the new token as the users `session_token` cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   newSessionToken,
		Expires: time.Now().Add(1800 * time.Second),
	})
	return userdata(username)
}

// func Logout(w http.ResponseWriter, r *http.Request) {
// 	c, err := r.Cookie("session_token")
// 	if err != nil {
// 		if err == http.ErrNoCookie {
// 			// If the cookie is not set, return an unauthorized status
// 			w.WriteHeader(http.StatusUnauthorized)
// 			http.Redirect(w, r, "/login?messagetitle=User Unauthorized&messagebody=Please login to use this site", http.StatusSeeOther)
// 			return
// 		}
// 		// For any other type of error, return a bad request status
// 		w.WriteHeader(http.StatusBadRequest)
// 		http.Redirect(w, r, "/login?messagetitle=User Unauthorized&messagebody=Please login to use this site", http.StatusSeeOther)
// 		return
// 	}
// 	sessionToken := c.Value

// 	// remove the users session from the session map
// 	delete(sessions, sessionToken)

// 	// We need to let the client know that the cookie is expired
// 	// In the response, we set the session token to an empty
// 	// value and set its expiry as the current time
// 	http.SetCookie(w, &http.Cookie{
// 		Name:    "session_token",
// 		Value:   "",
// 		Expires: time.Now(),
// 	})
// 	//Redirect Login
// 	http.Redirect(w, r, "/login?messagetitle=Logout Successful&messagebody=You have successfully been logged out", http.StatusSeeOther)
// }

func Logout(w http.ResponseWriter, r *http.Request) {
	// Get the session token from the user's cookies
	sessionToken, err := r.Cookie("session_token")
	if err != nil {
		// If the user doesn't have a session token cookie, they're already logged out
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	_, delErr := db.Exec("DELETE FROM sessions WHERE token = ?", sessionToken.Value)
	if delErr != nil {
		log.WithFields(log.Fields{"error": delErr}).Error("Error deleting session from database")
		// Handle error appropriately
	}

	// Remove the session token from the sessions map
	// delete(sessions, sessionToken.Value)

	// Set the session token cookie to expire immediately
	// http.SetCookie(w, &http.Cookie{
	// 	Name:    "session_token",
	// 	Value:   "",
	// 	Expires: time.Unix(0, 0),
	// })

	// // Set a cookie with the logout message
	// http.SetCookie(w, &http.Cookie{
	// 	Name:  "message",
	// 	Value: "You have been logged out",
	// 	Path:  "/login",
	// })

	// Redirect the user to the login page
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
