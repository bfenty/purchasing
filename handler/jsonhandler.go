package handler

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"purchasing/models"
	"time"

	log "github.com/sirupsen/logrus"
)

// respondWithJSON sends a JSON response to the client.
// It logs and handles errors that occur during JSON marshaling or writing to the response.
func RespondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	// Convert the payload to a JSON string
	response, err := json.Marshal(payload)
	if err != nil {
		// Log the error with the payload that caused it
		log.WithFields(log.Fields{
			"error":   err,
			"payload": payload,
		}).Error("Error marshalling JSON")

		// Respond with an Internal Server Error status and message
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error: Error marshalling JSON"))
		return
	}

	// Set the Content-Type of the response to application/json
	w.Header().Set("Content-Type", "application/json")

	// Write the status code to the response
	w.WriteHeader(statusCode)

	// Write the JSON response
	_, writeErr := w.Write(response)
	if writeErr != nil {
		// Log the error that occurred while writing the response
		log.WithFields(log.Fields{
			"error":   writeErr,
			"payload": payload,
		}).Error("Error writing JSON response")
	}

	// Log the successful response
	log.WithFields(log.Fields{
		"statusCode": statusCode,
		"response":   string(response),
	}).Debug("JSON response sent successfully")
}

// jsonLoad retrieves product data from the specified URL and unmarshals it into a 'product' struct.
// It makes an HTTP GET request to the provided URL, reads the response, and parses the JSON data.
// The function logs various stages of execution for debugging purposes.
func JsonLoad(url string) (products models.ProductBC) {
	log.Debug("Loading JSON from URL: ", url)

	// Define the HTTP client with a timeout
	commerceClient := http.Client{
		Timeout: time.Second * 20, // Timeout after 20 seconds
	}

	// Create an HTTP GET request
	log.Debug("Creating HTTP GET request")
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "url": url}).Error("Failed to create HTTP request")
	}

	// Setup the request header
	req.Header.Set("User-Agent", "commerce-client")
	req.Header.Add("x-auth-token", os.Getenv("BIGCOMMERCE_TOKEN"))

	// Perform the HTTP request
	res, getErr := commerceClient.Do(req)
	if getErr != nil {
		log.WithFields(log.Fields{"error": getErr, "url": url}).Error("Failed to execute HTTP request")
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	// Read the response body
	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.WithFields(log.Fields{"error": readErr}).Error("Failed to read response body")
	}

	log.Debug("Response Body: ", string(body))

	// Unmarshal the JSON response into 'products'
	products = models.ProductBC{}
	jsonErr := json.Unmarshal(body, &products)
	if jsonErr != nil {
		log.WithFields(log.Fields{"error": jsonErr}).Error("Failed to unmarshal JSON")
	}
	log.Debug("Retrieved Products: ", products)
	return products
}
