package api

import (
	"encoding/json"
	"net/http"
	"purchasing/config"
	_ "purchasing/docs"
	"purchasing/models"

	log "github.com/sirupsen/logrus"
)

// ListManufacturers godoc
//	@Summary		List manufacturers
//	@Description	Retrieves a list of manufacturers from the database
//	@Tags			manufacturers
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			Authorization	header		string				true	"Authorization"
//	@Success		200				{array}		models.Manufacturer	"List of manufacturers"
//	@Failure		500				{object}	map[string]string	"Internal Server Error"
//	@Router			/api/manufacturers [get]

func ListManufacturers(w http.ResponseWriter, r *http.Request) {
	rows, err := config.DB.Query("SELECT name FROM purchasing.manufacturers WHERE 1")
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error executing query")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var manufacturers []models.Manufacturer
	for rows.Next() {
		var m models.Manufacturer
		if err := rows.Scan(&m.Name); err != nil {
			log.WithFields(log.Fields{"error": err}).Error("Error scanning row")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		manufacturers = append(manufacturers, m)
	}

	// Handle any error encountered during iteration
	if err = rows.Err(); err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error iterating over rows")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Set header and encode the result into JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(manufacturers); err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error encoding JSON")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
