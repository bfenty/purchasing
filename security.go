package main

import (
	"fmt"
	// "log"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

func getPwd() []byte { // Prompt the user to enter a password
	fmt.Println("Enter a password") // We will use this to store the users input
	var pwd string                  // Read the users input
	_, err := fmt.Scan(&pwd)
	if err != nil {
		handleerror(err)
	} // Return the users input as a byte slice which will save us
	// from having to do this conversion later on
	return []byte(pwd)
}

func hashAndSalt(pwd []byte) string {

	// Use GenerateFromPassword to hash & salt pwd
	// MinCost is just an integer constant provided by the bcrypt
	// package along with DefaultCost & MaxCost.
	// The cost can be any value you want provided it isn't lower
	// than the MinCost (4)
	hash, err := bcrypt.GenerateFromPassword(pwd, bcrypt.MinCost)
	if err != nil {
		handleerror(err)
	} // GenerateFromPassword returns a byte slice so we need to
	// convert the bytes to a string and return it
	return string(hash)
}

func comparePasswords(hashedPwd string, plainPwd []byte) bool {
	// Since we'll be getting the hashed password from the DB it
	// will be a string so we'll need to convert it to a byte slice
	// Log.Debug("Comparing passwords: ",plainPwd, hashedPwd)
	byteHash := []byte(hashedPwd)
	err := bcrypt.CompareHashAndPassword(byteHash, plainPwd)
	if err != nil {
		handleerror(err)
		log.Debug("Passwords do not match")
		return false
	}
	log.Debug("Passwords match")
	return true
}
