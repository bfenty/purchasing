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
	log.Debug("Entering Signin function")

	// Parse form values directly from the HTTP request
	if err := r.ParseForm(); err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error parsing form")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var creds Credentials
	creds.Username = r.FormValue("username")
	creds.Password = r.FormValue("password")

	// Debugging credentials (avoid logging sensitive data like passwords in production)
	log.WithFields(log.Fields{
		"username": creds.Username,
	}).Debug("Received credentials")

	// Authenticating user
	user, message := userauth(creds.Username, creds.Password)
	log.WithFields(log.Fields{
		"username":    user.Username,
		"role":        user.Role,
		"authMessage": message.Body,
	}).Debug("User authentication result")

	// Handle different authentication outcomes
	switch user.Role {
	case "notfound":
		log.WithFields(log.Fields{
			"username": creds.Username,
			"error":    message.Body,
		}).Debug("Authentication failed: User not found or invalid credentials")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, message.Body)
		return

	case "newuser":
		log.WithFields(log.Fields{
			"username": creds.Username,
		}).Debug("Redirecting new user to signup page")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, "/signup")
		return

	default:

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

		// Setting client cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    sessionToken,
			Expires:  expiresAt,
			Path:     "/",  // Setting the path to root
			HttpOnly: true, // Recommended to mitigate the risk of client side script accessing the protected cookie
		})

		log.WithFields(log.Fields{
			"username":     user.Username,
			"sessionToken": sessionToken,
		}).Debug("Session token created and cookie set")

		// Handling redirects based on role
		redirectURL := "/products?layout=insert" // Default redirect
		if user.Role == "sorting" {
			redirectURL = "/checkout"
		} else if user.Role == "receiving" {
			redirectURL = "/products"
		}

		log.WithFields(log.Fields{
			"username":    user.Username,
			"role":        user.Role,
			"redirectURL": redirectURL,
		}).Debug("Redirecting user based on role")

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, redirectURL)
		return
	}
}

func auth(w http.ResponseWriter, r *http.Request) (user User) {
	log.Debug("Entering auth function")

	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			log.Debug("No session token cookie found, redirecting to login")
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return User{Role: "Unauthorized"} // Return immediately after redirect
		}
		log.Error("Error retrieving session token cookie:", err)
		return User{Role: "Unauthorized"}
	}
	log.Debug("Session token cookie found:", c.Value)
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
	err = db.QueryRow("SELECT username, expiry FROM purchasing.sessions WHERE token = ?", sessionToken).Scan(&username, &expiry)
	if err != nil {
		// Handle no rows found or other errors
		log.Debug("Session token not found in database, redirecting to login")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return User{Role: "Unauthorized"}
	}

	if expiry.Before(time.Now()) {
		// Session is expired, delete it and redirect to login
		_, delErr := db.Exec("DELETE FROM purchasing.sessions WHERE token = ?", sessionToken)
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
	_, delErr := db.Exec("DELETE FROM purchasing.sessions WHERE token = ?", sessionToken)
	if delErr != nil {
		log.WithFields(log.Fields{"error": delErr}).Error("Error deleting expired session")
	}

	// Replace the session map assignment with a database insert
	_, err = db.Exec("INSERT INTO purchasing.sessions (token, username, expiry) VALUES (?, ?, ?)", newSessionToken, username, expiresAt)
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

	log.Debug("New session token created:", newSessionToken)
	delete(sessions, sessionToken)

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

func userauth(username string, pass string) (user User, message Message) {
	log.WithFields(log.Fields{
		"username": username,
	}).Debug("Entering userauth function")

	// Test database connection
	pingErr := db.Ping()
	if pingErr != nil {
		log.WithFields(log.Fields{
			"error": pingErr,
		}).Error("Database connection error")
		user.Role = "notfound"
		return user, handleerror(pingErr)
	}

	user.Username = username
	var dbpass string
	var newquery string = "select password, permissions, admin, management from orders.users where username = ?"

	rows, err := db.Query(newquery, username)
	if err != nil {
		log.WithFields(log.Fields{
			"username": username,
			"error":    err,
		}).Error("Error executing database query")
		user.Role = "notfound"
		return user, handleerror(err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&dbpass, &user.Role, &user.Permissions.Admin, &user.Permissions.Mgmt)
		if err != nil {
			log.WithFields(log.Fields{
				"username": username,
				"error":    err,
			}).Error("Error scanning database rows")
			user.Role = "notfound"
			return user, handleerror(err)
		}
		log.WithFields(log.Fields{
			"username": username,
			"role":     user.Role,
		}).Debug("User data retrieved from database")
	}

	if err = rows.Err(); err != nil {
		log.WithFields(log.Fields{
			"username": username,
			"error":    err,
		}).Error("Error in database rows")
		user.Role = "notfound"
		return user, handleerror(err)
	}

	if user.Role == "" {
		message.Title = "Permission not found"
		message.Body = "Permissions not set for user. Please contact your system administrator."
		log.WithFields(log.Fields{
			"username": username,
		}).Debug("User role not found")
		user.Role = "notfound"
		return user, message
	}

	if dbpass == "" {
		message.Title = "Set Password"
		message.Body = "Password not set, please create password"
		log.WithFields(log.Fields{
			"username": username,
		}).Debug("User password not set")
		user.Role = "newuser"
		return user, message
	}

	if comparePasswords(dbpass, []byte(pass)) {
		message.Title = "Success"
		message.Body = "Successfully logged in"
		message.Success = true
		log.WithFields(log.Fields{
			"username": username,
		}).Debug("User authenticated successfully")
		return user, message
	}

	message.Title = "Login Failed"
	message.Body = "Login Failed"
	log.WithFields(log.Fields{
		"username": username,
	}).Debug("User authentication failed")
	user.Role = "notfound"
	return user, message
}

func userdata(username string) (user User) {
	log.WithFields(log.Fields{
		"username": username,
	}).Debug("Entering userdata function")

	// Test database connection
	pingErr := db.Ping()
	if pingErr != nil {
		log.WithFields(log.Fields{
			"error": pingErr,
		}).Error("Database connection error")
		handleerror(pingErr)
		return user
	}

	var newquery string = "select username, usercode, permissions, admin, management, manager, sorting from orders.users where username = ?"
	log.WithFields(log.Fields{
		"query": newquery,
	}).Debug("Executing database query")

	rows, err := db.Query(newquery, username)
	if err != nil {
		log.WithFields(log.Fields{
			"username": username,
			"error":    err,
		}).Error("Error executing database query")
		handleerror(err)
		return user
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&user.Username, &user.Usercode, &user.Role, &user.Permissions.Admin, &user.Permissions.Mgmt, &user.Manager, &user.Permissions.Sorting)
		if err != nil {
			log.WithFields(log.Fields{
				"username": username,
				"error":    err,
			}).Error("Error scanning database rows")
			handleerror(err)
			return user
		}
		log.WithFields(log.Fields{
			"username": username,
			"role":     user.Role,
		}).Debug("User data retrieved from database")
	}

	if err = rows.Err(); err != nil {
		log.WithFields(log.Fields{
			"username": username,
			"error":    err,
		}).Error("Error in database rows")
		handleerror(err)
		return user
	}

	if user.Role == "" {
		log.WithFields(log.Fields{
			"username": username,
		}).Debug("User role not found, setting role to 'notfound'")
		user.Role = "notfound"
	}

	log.WithFields(log.Fields{
		"username": username,
		"role":     user.Role,
	}).Debug("Exiting userdata function")
	return user
}
