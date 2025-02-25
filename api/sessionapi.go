package api

import (
	"fmt"
	"net/http"
	"purchasing/config"
	"purchasing/models"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// Signin godoc
//	@Summary		User login
//	@Description	Authenticates a user and returns a session token as a cookie
//	@Tags			auth
//	@Accept			application/x-www-form-urlencoded
//	@Produce		text/plain
//	@Param			username	formData	string				true	"Username"
//	@Param			password	formData	string				true	"Password"
//	@Success		200			{string}	string				"Redirect URL based on user role"
//	@Success		201			{string}	string				"Redirect to /signup for new users"
//	@Failure		400			{object}	map[string]string	"Bad Request: Error parsing form"
//	@Failure		401			{object}	map[string]string	"Unauthorized: Invalid credentials"
//	@Failure		500			{object}	map[string]string	"Internal Server Error"
//	@Router			/api/signin [post]

func Signin(w http.ResponseWriter, r *http.Request) {
	log.Debug("Entering Signin function")

	// Parse form values directly from the HTTP request
	if err := r.ParseForm(); err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error parsing form")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var creds models.Credentials
	creds.Username = r.FormValue("username")
	creds.Password = r.FormValue("password")

	// Debugging credentials (avoid logging sensitive data like passwords in production)
	log.WithFields(log.Fields{
		"username": creds.Username,
	}).Debug("Received credentials")

	// Authenticating user
	user, message := Userauth(creds.Username, creds.Password)
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
		// use the "github.com/google/uuid" library to generate UUIDs
		sessionToken := uuid.NewString()
		expiresAt := time.Now().Add(1800 * time.Second)
		// log.Debug("Authorized")

		// Set the token in the session map, along with the session information
		models.Sessions[sessionToken] = models.Session{
			Username: creds.Username,
			Expiry:   expiresAt,
		}

		// Replace the session map assignment with a database insert
		_, err := config.DB.Exec("INSERT INTO purchasing.sessions (token, username, expiry) VALUES (?, ?, ?)", sessionToken, creds.Username, expiresAt)
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
			Path:     "/", // Setting the path to root
			HttpOnly: true,
		})

		log.WithFields(log.Fields{
			"username":     user.Username,
			"sessionToken": sessionToken,
		}).Debug("Session token created and cookie set")

		// Handling redirects based on role
		redirectURL := "/products?layout=insert" // Default redirect
		if user.Role == "sorting" {
			redirectURL = "/sorting?layout=checkout"
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

// Logout godoc
//	@Summary		User logout
//	@Description	Logs out the user by deleting the session token
//	@Tags			auth
//	@Accept			json
//	@Produce		text/plain
//	@Success		303	{string}	string				"Redirect to /login"
//	@Failure		500	{object}	map[string]string	"Internal Server Error"
//	@Router			/api/logout [get]

func Logout(w http.ResponseWriter, r *http.Request) {
	// Get the session token from the user's cookies
	sessionToken, err := r.Cookie("session_token")
	if err != nil {
		// If the user doesn't have a session token cookie, they're already logged out
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	_, delErr := config.DB.Exec("DELETE FROM sessions WHERE token = ?", sessionToken.Value)
	if delErr != nil {
		log.WithFields(log.Fields{"error": delErr}).Error("Error deleting session from database")
		// Handle error appropriately
	}

	// Redirect the user to the login page
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
