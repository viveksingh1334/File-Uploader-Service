package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

var maxSize int64 = 10 << 20 // 10 MB limit for file size

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", homeHandler).Methods("GET")
	r.HandleFunc("/", uploadFileHandler).Methods("POST")
	r.HandleFunc("/login", loginHandler).Methods("POST")
	r.HandleFunc("/logout", logoutHandler).Methods("POST")
	r.HandleFunc("/download/{filename}", downloadHandler).Methods("GET")
	r.HandleFunc("/delete/{filename}", deleteHandler).Methods("POST")

	http.Handle("/", r)
	http.ListenAndServe(":8080", nil)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	// Check if the user is authenticated
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// List all the uploaded files for the user
	files := listUserFiles(r)
	renderTemplate(w, "index.html", files)
}

func uploadFileHandler(w http.ResponseWriter, r *http.Request) {
	// Check if the user is authenticated
	if !isLoggedIn(r) {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Parse the multipart form
	err := r.ParseMultipartForm(maxSize)
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	// Get the file from the form
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Check the file size
	if handler.Size > maxSize {
		http.Error(w, "File size exceeds the limit", http.StatusBadRequest)
		return
	}

	// Create a destination file on the server
	username := getUsername(r)
	dstPath := filepath.Join("uploads", username, handler.Filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "Error creating destination file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy the uploaded file to the destination file on the server
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Error copying file", http.StatusInternalServerError)
		return
	}

	// Redirect the user back to the home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	// Check if the user is authenticated
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get the filename from the URL path parameter
	vars := mux.Vars(r)
	filename := vars["filename"]

	// Check if the file exists
	username := getUsername(r)
	filePath := filepath.Join("uploads", username, filename)
	if _, err := os.Stat(filePath); err != nil {
		http.NotFound(w, r)
		return
	}

	// Set the appropriate Content-Type header
	ext := strings.ToLower(filepath.Ext(filename))
	contentType := "application/octet-stream"
	if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" {
		contentType = "image/" + ext[1:]
	} else if ext == ".pdf" {
		contentType = "application/pdf"
	}
	w.Header().Set("Content-Type", contentType)

	// Serve the file for download
	http.ServeFile(w, r, filePath)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	// Check if the user is authenticated
	if !isLoggedIn(r) {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Get the filename from the URL path parameter
	vars := mux.Vars(r)
	filename := vars["filename"]

	// Check if the file exists
	username := getUsername(r)
	filePath := filepath.Join("uploads", username, filename)
	if _, err := os.Stat(filePath); err != nil {
		http.NotFound(w, r)
		return
	}

	// Delete the file
	err := os.Remove(filePath)
	if err != nil {
		http.Error(w, "Error deleting file", http.StatusInternalServerError)
		return
	}

	// Redirect the user back to the home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
