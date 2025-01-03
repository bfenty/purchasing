package config

import (
	"database/sql"
	"os"

	// "purchasing/handler"
	"purchasing/handler"
	"purchasing/models"

	log "github.com/sirupsen/logrus"
)

var DB *sql.DB

// Opens the databse connection for the application
func Opendb() (db *sql.DB, messagebox models.Message) {
	var err error
	user := os.Getenv("USER")
	pass := os.Getenv("PASS")
	server := os.Getenv("SERVER")
	port := os.Getenv("PORT")
	// Get a database handle.
	log.Info("Connecting to DB...")
	log.Debug("user:", user)
	log.Debug("pass:", pass)
	log.Debug("server:", server)
	log.Debug("port:", port)
	log.Debug("Opening Database...")
	connectstring := os.Getenv("USER") + ":" + os.Getenv("PASS") + "@tcp(" + os.Getenv("SERVER") + ":" + os.Getenv("PORT") + ")/?parseTime=true"
	log.Debug("Connection: ", connectstring)
	db, err = sql.Open("mysql",
		connectstring)
	if err != nil {
		messagebox.Success = false
		messagebox.Body = err.Error()
		log.Debug("Message: ", messagebox.Body)
		return nil, messagebox
	}

	//Test Connection
	pingErr := db.Ping()
	if pingErr != nil {
		return nil, handler.Handleerror(pingErr)
	}

	//Success!
	log.Info("Returning Open DB...")
	messagebox.Success = true
	messagebox.Body = "Success"
	return db, messagebox
}
