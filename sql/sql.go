package sql

import (

	// "log"

	"purchasing/config"
	"purchasing/handler"
	"purchasing/models"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

type Request struct {
	Sorter      string `json:"sorter"`
	Description string `json:"description"`
}

type ApiResponse struct {
	Message string `json:"message"`
}

func ProductExistSQL(sku string) (exists string, message models.Message) {
	log.Info("SKU: ", sku)
	//Test Connection
	pingErr := config.DB.Ping()
	if pingErr != nil {
		config.DB, message = config.Opendb()
		return exists, handler.Handleerror(pingErr)
	}

	// sku = "TEST"
	newquery := "SELECT COUNT(*) from skus where sku_internal = ?"
	var count int
	err := config.DB.QueryRow(newquery, sku).Scan(&count)
	if err != nil {
		handler.Handleerror(err)
	}
	log.Info("Count: ", count)

	if count == 0 {
		exists = "FALSE"
	} else {
		exists = "TRUE"
	}

	return exists, message
}

// func userUpdateHandler(w http.ResponseWriter, r *http.Request) {

// 	//Test Connection
// 	pingErr := config.DB.Ping()
// 	if pingErr != nil {
// 		config.DB, _ = config.Opendb()
// 		// return handler.Handleerror(pingErr)
// 	}

// 	// Get the form values from the request
// 	username := r.FormValue("username")
// 	usercode := r.FormValue("usercode")
// 	role := r.FormValue("role")
// 	manager := r.FormValue("manager")
// 	var sorting int
// 	if r.FormValue("sorting") == "true" {
// 		sorting = 1
// 	} else {
// 		sorting = 0
// 	}
// 	// sorting, _ := strconv.ParseBool(r.FormValue("sorting"))
// 	println(username, usercode, role, sorting)

// 	// If usercode is empty, find the current max value and increment it
// 	if usercode == "" {
// 		var maxUsercode int
// 		err := config.DB.QueryRow("SELECT MAX(usercode) FROM orders.users").Scan(&maxUsercode)
// 		if err != nil {
// 			// Handle error
// 			handler.Handleerror(err)
// 		}
// 		usercode = strconv.Itoa(maxUsercode + 1)
// 	}

// 	// Prepare the SQL statement for inserting the data
// 	//Logging
// 	log.Info("Creating Query")
// 	newquery := "REPLACE INTO orders.users (username, usercode, permissions, sorting,manager) VALUES (?, ?, ?, ?, ?)"

// 	// Execute the SQL statement with the form values
// 	log.Info("Executing Query")
// 	rows, err := config.DB.Query(newquery, username, usercode, role, sorting, manager)
// 	defer rows.Close()

// 	if err != nil {
// 		// Handle error
// 		println(err)
// 	}

// 	// Redirect the user to the users page
// 	http.Redirect(w, r, "/users", http.StatusSeeOther)
// }

// func userDeleteHandler(w http.ResponseWriter, r *http.Request) {
// 	// Get the usercode value from the form
// 	usercode := r.FormValue("usercode")

// 	// Prepare the SQL statement for deleting the user
// 	stmt, err := config.DB.Prepare("UPDATE orders.users SET active = 0 WHERE usercode = ?")
// 	if err != nil {
// 		// Handle error
// 		println(err)
// 		http.Error(w, "Failed to update user information.", http.StatusInternalServerError)
// 		return
// 	}
// 	defer stmt.Close()

// 	// Execute the SQL statement with the usercode value
// 	_, err = stmt.Exec(usercode)
// 	if err != nil {
// 		// Handle error
// 		println(err)
// 		http.Error(w, "Failed to update user information.", http.StatusInternalServerError)
// 		return
// 	}

// 	// Redirect the user to the users page
// 	w.Write([]byte("User information updated successfully."))
// }

// func listorders(user User) (message Message, orders []Order) {
// 	//Debug
// 	log.WithFields(log.Fields{"username": user.Username}).Debug("Getting Orders...")

// 	//Test Connection
// 	pingErr := config.DB.Ping()
// 	if pingErr != nil {
// 		config.DB, message = config.Opendb()
// 		return handler.Handleerror(pingErr), orders
// 	}
// 	//Build the Query
// 	newquery := "SELECT ordernum,trackingnum,comments,manufacturer,status FROM `orders` WHERE 1"

// 	//Run Query
// 	rows, err := config.DB.Query(newquery)
// 	defer rows.Close()
// 	if err != nil {
// 		return handler.Handleerror(err), orders
// 	}

// 	//Pull Data
// 	for rows.Next() {
// 		var r Order
// 		err := rows.Scan(&r.Ordernum, &r.Tracking, &r.Comments, &r.Manufacturer, &r.Status)
// 		if err != nil {
// 			return handler.Handleerror(err), orders
// 		}
// 		orders = append(orders, r)
// 	}
// 	return message, orders
// }

func nextorder(manufacturer string, user models.User) (message models.Message, order models.Order) {
	var ordernum int
	pingErr := config.DB.Ping()
	if pingErr != nil {
		config.DB, message = config.Opendb()
		return handler.Handleerror(pingErr), order
	}
	//Build the Query
	newquery := "SELECT MAX(ordernum) ordernum  FROM `orders` WHERE 1"

	rows, err := config.DB.Query(newquery)
	defer rows.Close()
	if err != nil {
		return handler.Handleerror(err), order
	}
	// var val string
	if rows.Next() {
		rows.Scan(&ordernum)
	}
	ordernum += 1
	log.WithFields(log.Fields{"username": user.Username}).Debug(manufacturer, "-", ordernum)

	//insert new order into database
	newquery = "INSERT INTO orders (`ordernum`,`manufacturer`) VALUES (?,?)"
	orderinsert, err := config.DB.Query(newquery, ordernum, manufacturer)
	orderinsert.Close()
	message.Success = true
	message.Body = "Successfully created order " + manufacturer + "-" + strconv.Itoa(ordernum)
	order.Ordernum = ordernum
	return message, order
}

func sortrequestdeletesql(requestid int, user models.User) (message models.Message) {
	//Debug
	log.WithFields(log.Fields{"username": user.Username}).Info("Deleting order ", "...")

	//Test Connection
	pingErr := config.DB.Ping()
	if pingErr != nil {
		config.DB, message = config.Opendb()
		return handler.Handleerror(pingErr)
	}

	//Build the Query
	newquery := "DELETE FROM `sortrequest` WHERE requestid = ?"
	rows, err := config.DB.Query(newquery, requestid)
	rows.Close()
	if err != nil {
		return handler.Handleerror(err)
	}

	message.Success = true
	message.Title = "Success"
	message.Body = "Successfully deleted Sorting Request ID  " + strconv.Itoa(requestid)
	return message
}
