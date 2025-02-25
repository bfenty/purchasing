package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"purchasing/config"
	"purchasing/handler"
	"purchasing/models"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

// ListUsersAPI godoc
//	@Summary		List users
//	@Description	Retrieves a list of users with optional filtering parameters
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			username	query		string					false	"Filter users by username"
//	@Param			usercode	query		string					false	"Filter users by usercode"
//	@Param			role		query		string					false	"Filter users by role (e.g., 'sorting', 'manager')"
//	@Param			sorting		query		string					false	"Filter by sorting permissions"
//	@Param			manager		query		string					false	"Filter by manager permissions"
//	@Param			management	query		string					false	"Filter by management permissions"
//	@Param			active		query		string					false	"Filter by active status (1 for active, 0 for inactive)"
//	@Param			page		query		int						false	"Page number (default: 1)"
//	@Param			limit		query		int						false	"Number of records per page (default: 100, max: 100)"
//	@Success		200			{object}	map[string]interface{}	"List of users with pagination"
//	@Failure		500			{object}	map[string]string		"Internal Server Error"
//	@Router			/api/users [get]

func ListUsersAPI(w http.ResponseWriter, r *http.Request) {
	log := logrus.WithFields(logrus.Fields{
		"api":    "ListUsersAPI",
		"method": r.Method,
		"path":   r.URL.Path,
	})

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 { // set a maximum limit to prevent abuse
		limit = 100
	}

	searchParams := map[string]string{
		"username":   r.URL.Query().Get("username"),
		"usercode":   r.URL.Query().Get("usercode"),
		"role":       r.URL.Query().Get("role"),
		"sorting":    r.URL.Query().Get("sorting"),
		"manager":    r.URL.Query().Get("manager"),
		"management": r.URL.Query().Get("management"),
		"active":     r.URL.Query().Get("active"),
	}

	var queryParams []interface{}
	var whereClauses []string
	for key, value := range searchParams {
		if value != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", key))
			queryParams = append(queryParams, value)
		}
	}

	whereClause := strings.Join(whereClauses, " AND ")
	if whereClause != "" {
		whereClause = " AND " + whereClause
	}

	query := fmt.Sprintf("SELECT username, usercode, permissions, sorting, manager, management FROM orders.users WHERE 1=1%s LIMIT ? OFFSET ?", whereClause)
	queryParams = append(queryParams, limit, (page-1)*limit)

	log.WithField("query", query).Debug("Executing user query")

	rows, err := config.DB.Query(query, queryParams...)
	if err != nil {
		log.WithError(err).Error("Failed to execute user query")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.Username, &user.Usercode, &user.Role, &user.Sorting, &user.Manager, &user.Management); err != nil {
			log.WithError(err).Error("Failed to scan user row")
			continue
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		log.WithError(err).Error("Error iterating user rows")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"users":       users,
		"currentPage": page,
		"totalPages":  (len(users) + limit - 1) / limit, // Simplified pagination logic
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.WithError(err).Error("Failed to encode response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	log.WithFields(logrus.Fields{
		"numUsers":    len(users),
		"currentPage": page,
		"totalPages":  (len(users) + limit - 1) / limit,
	}).Info("Successfully retrieved user list")
}

// Usercreate godoc
//	@Summary		Create a new user
//	@Description	Registers a new user in the system
//	@Tags			users
//	@Accept			application/x-www-form-urlencoded
//	@Produce		text/plain
//	@Param			username	formData	string				true	"Username of the new user"
//	@Param			password	formData	string				true	"Password for the new user"
//	@Param			password2	formData	string				true	"Confirmation of the password"
//	@Param			secret		formData	string				true	"Secret key for authentication"
//	@Success		303			{string}	string				"Redirect to /products if successful, otherwise to /signup with an error message"
//	@Failure		400			{object}	map[string]string	"Bad Request: Invalid input or passwords do not match"
//	@Router			/api/usercreate [post]

func Usercreate(w http.ResponseWriter, r *http.Request) {
	var creds models.Credentials
	var message models.Message
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

func Auth(w http.ResponseWriter, r *http.Request) (user models.User) {
	log.Debug("Entering auth function")

	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			log.Debug("No session token cookie found, redirecting to login")
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return models.User{Role: "Unauthorized"} // Return immediately after redirect
		}
		log.Error("Error retrieving session token cookie:", err)
		return models.User{Role: "Unauthorized"}
	}
	log.Debug("Session token cookie found:", c.Value)
	sessionToken := c.Value

	var username string
	var expiry time.Time
	err = config.DB.QueryRow("SELECT username, expiry FROM purchasing.sessions WHERE token = ?", sessionToken).Scan(&username, &expiry)
	if err != nil {
		// Handle no rows found or other errors
		log.Debug("Session token not found in database, redirecting to login")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return models.User{Role: "Unauthorized"}
	}

	if expiry.Before(time.Now()) {
		// Session is expired, delete it and redirect to login
		_, delErr := config.DB.Exec("DELETE FROM purchasing.sessions WHERE token = ?", sessionToken)
		if delErr != nil {
			log.WithFields(log.Fields{"error": delErr}).Error("Error deleting expired session")
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return models.User{Role: "Unauthorized"}
	}

	// Finally, return the welcome message to the user
	log.Debug("Authorized")
	// If the previous session is valid, create a new session token for the current user
	newSessionToken := uuid.NewString()
	expiresAt := time.Now().Add(1800 * time.Second)

	// First, delete the old session ID
	_, delErr := config.DB.Exec("DELETE FROM purchasing.sessions WHERE token = ?", sessionToken)
	if delErr != nil {
		log.WithFields(log.Fields{"error": delErr}).Error("Error deleting expired session")
	}

	// Replace the session map assignment with a database insert
	_, err = config.DB.Exec("INSERT INTO purchasing.sessions (token, username, expiry) VALUES (?, ?, ?)", newSessionToken, username, expiresAt)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error saving session to database")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Debug("New session token created:", newSessionToken)
	delete(models.Sessions, sessionToken)

	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   newSessionToken,
		Expires: time.Now().Add(1800 * time.Second),
	})
	return userdata(username)
}

func Userauth(username string, pass string) (user models.User, message models.Message) {
	log.WithFields(log.Fields{
		"username": username,
	}).Debug("Entering userauth function")

	// Test database connection
	pingErr := config.DB.Ping()
	if pingErr != nil {
		log.WithFields(log.Fields{
			"error": pingErr,
		}).Error("Database connection error")
		user.Role = "notfound"
		return user, handler.Handleerror(pingErr)
	}

	user.Username = username
	var dbpass string
	var newquery string = "select password, permissions, admin, management from orders.users where username = ?"

	rows, err := config.DB.Query(newquery, username)
	if err != nil {
		log.WithFields(log.Fields{
			"username": username,
			"error":    err,
		}).Error("Error executing database query")
		user.Role = "notfound"
		return user, handler.Handleerror(err)
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
			return user, handler.Handleerror(err)
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
		return user, handler.Handleerror(err)
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

	if handler.ComparePasswords(dbpass, []byte(pass)) {
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

func userdata(username string) (user models.User) {
	log.WithFields(log.Fields{
		"username": username,
	}).Debug("Entering userdata function")

	// Test database connection
	pingErr := config.DB.Ping()
	if pingErr != nil {
		log.WithFields(log.Fields{
			"error": pingErr,
		}).Error("Database connection error")
		handler.Handleerror(pingErr)
		return user
	}

	var newquery string = "select username, usercode, permissions, admin, management, manager, sorting from orders.users where username = ?"
	log.WithFields(log.Fields{
		"query": newquery,
	}).Debug("Executing database query")

	rows, err := config.DB.Query(newquery, username)
	if err != nil {
		log.WithFields(log.Fields{
			"username": username,
			"error":    err,
		}).Error("Error executing database query")
		handler.Handleerror(err)
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
			handler.Handleerror(err)
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
		handler.Handleerror(err)
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

// Update User Password
func Updatepass(user string, pass string, secret string) (message models.Message, success bool) {
	pingErr := config.DB.Ping()
	if pingErr != nil {
		return handler.Handleerror(pingErr), false
	}

	//Check for secret
	if secret != os.Getenv("SECRET") {
		message.Title = "Secret Auth Failed"
		message.Body = "Secret Auth Failed"
		return message, false
	}

	hashpass := handler.HashAndSalt([]byte(pass))
	log.Debug("Creating password hash of length ", len(hashpass), ": ", hashpass)
	var newquery string = "update orders.users set password = ? where username = ? and password = ''"
	rows, err := config.DB.Query(newquery, hashpass, user)
	if err != nil {
		return handler.Handleerror(err), false
	}
	defer rows.Close()
	message.Title = "Success"
	message.Body = "Success"
	message.Success = true
	return message, true
}

// UserUpdateAPI godoc
//	@Summary		Update user details
//	@Description	Updates user information such as role, sorting permissions, and management status
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			user	body		models.User			true	"User details to update"
//	@Success		200		{object}	map[string]string	"User information updated successfully"
//	@Failure		400		{object}	map[string]string	"Bad Request: Invalid request data"
//	@Failure		500		{object}	map[string]string	"Internal Server Error"
//	@Router			/api/userupdate [post]

func UpdateUser(w http.ResponseWriter, r *http.Request) {

	//Test Connection
	pingErr := config.DB.Ping()
	if pingErr != nil {
		config.DB, _ = config.Opendb()
		// return handler.Handleerror(pingErr)
	}

	user := Auth(w, r)

	// Get the form values from the request
	username := r.FormValue("username")
	usercode := r.FormValue("usercode")
	role := r.FormValue("role")
	manager := r.FormValue("manager")
	log.WithFields(log.Fields{"username": user.Username}).Debug("username:", username, " usercode:", usercode, " role:", role, " manager:", manager)
	var sorting int
	if r.FormValue("sorting") == "true" {
		sorting = 1
	} else {
		sorting = 0
	}
	var management int
	if r.FormValue("management") == "true" {
		management = 1
	} else {
		management = 0
	}
	// sorting, _ := strconv.ParseBool(r.FormValue("sorting"))
	println(username, usercode, role, sorting)

	// If usercode is empty, find the current max value and increment it
	if usercode == "" {
		var maxUsercode int
		err := config.DB.QueryRow("SELECT MAX(usercode) FROM orders.users").Scan(&maxUsercode)
		if err != nil {
			// Handle error
			handler.Handleerror(err)
		}
		usercode = strconv.Itoa(maxUsercode + 1)
	}

	// Prepare the SQL statement for inserting or updating the data
	newquery := `
		INSERT INTO orders.users (username, usercode, permissions, sorting, manager, management)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
		username = VALUES(username),
		permissions = VALUES(permissions),
		sorting = VALUES(sorting),
		manager = VALUES(manager),
		management = VALUES(management)
		`

	// Execute the SQL statement with the form values
	rows, err := config.DB.Query(newquery, username, usercode, role, sorting, manager, management)
	defer rows.Close()

	if err != nil {
		// Handle error
		println(err)
		http.Error(w, "Failed to update user information.", http.StatusInternalServerError)
		return
	}

	// Return success message to client
	w.Write([]byte("User information updated successfully."))
}

// UserDeleteAPI godoc
//	@Summary		Delete a user
//	@Description	Deactivates a user by setting their active status to 0
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			usercode	body		string				true	"Usercode of the user to deactivate"
//	@Success		200			{object}	map[string]string	"User successfully deactivated"
//	@Failure		400			{object}	map[string]string	"Bad Request: Usercode is required"
//	@Failure		404			{object}	map[string]string	"User not found"
//	@Failure		500			{object}	map[string]string	"Internal Server Error"
//	@Router			/api/userdelete [post]

func UserUpdateAPI(w http.ResponseWriter, r *http.Request) {
	var user models.User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error decoding user data")
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Internal Server Error: %v", err)})
		return
	}

	// SQL DELETE statement
	query := `UPDATE orders.users SET active=0 WHERE usercode=?`
	_, err = config.DB.Exec(query, user.Usercode)
	if err != nil {
		log.WithFields(log.Fields{"usercode": user.Usercode, "error": err}).Error("Error executing delete query")
		handler.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Internal Server Error: %v", err)})
		return
	}

	// Respond with success message
	handler.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "User successfully deleted"})
}
