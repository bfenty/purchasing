package handler

import (
	"net/http"
	"purchasing/models"

	log "github.com/sirupsen/logrus"
)

// Handle Messages
func message(r *http.Request) (messagebox models.Message) {
	if r.URL.Query().Get("messagetitle") != "" {
		messagebox.Body = r.URL.Query().Get("messagebody")
		log.Info("Message: ", messagebox)
	}
	return messagebox
}
