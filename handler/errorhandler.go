package handler

import (
	"encoding/json"
	"net/http"
	"purchasing/models"

	log "github.com/sirupsen/logrus"
)

func Handleerror2(err error, w http.ResponseWriter) models.Message {
	log.Error(err)
	message := models.Message{Title: "Error", Body: err.Error(), Success: false}

	// encode message as JSON
	response, _ := json.Marshal(message)

	// set content type to JSON and send response
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)

	// Log the response JSON for debugging
	// log.WithFields(log.Fields{
	// 	"responseJSON": string(response),
	// }).Debug("Error response sent to client")

	return message
}

// Handle Error Messages
func Handleerror(err error) (message models.Message) {
	if err != nil {
		message.Title = "Error"
		message.Success = false
		message.Body = err.Error()
		log.Error(message.Body)
		return message
	}
	message.Success = true
	message.Body = "Success"
	return message
}
