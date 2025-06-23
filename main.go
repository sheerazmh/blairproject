package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"      // For logging server messages to the console
	"net/http" // The core package for building web servers
)

type UserSignUpRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ModifyImageRequest struct {
	ImageID   string `json:"image_id"`
	ModPrompt string `json:"prompt"`
}

type GenericResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

// homeHandler serves the root path "/"
func homeHandler(w http.ResponseWriter, r *http.Request) {
	// fmt.Fprintf writes formatted output to the ResponseWriter (w)
	fmt.Fprintf(w, "Welcome to the Flowify API! Your creative asset transformation journey begins.")
	log.Println("Received GET request for /") // Log the request for debugging
}

// usersHandler manages user-related actions (signup, login, profile)
func usersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Ensure the request body is closed after we're done reading it
	defer r.Body.Close()

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		log.Printf("Error reading usersHandler body: %v", err)
		return
	}

	var req UserSignUpRequest
	// Unmarshal the JSON body into our UserSignUpRequest struct
	err = json.Unmarshal(body, &req)
	if err != nil {
		// If JSON is malformed, send a Bad Request error
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		log.Printf("Error unmarshaling usersHandler JSON: %v", err)
		return
	}

	// Simulate processing: In a real app, you'd save this to a database, hash password etc.
	log.Printf("Simulating new user signup: Email=%s, Password=%s", req.Email, req.Password)

	// Prepare a success response
	response := GenericResponse{
		Message: fmt.Sprintf("User '%s' registered successfully (simulated).", req.Email),
		Status:  "success",
	}

	// Set the Content-Type header to application/json
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // Send 200 OK status

	// Marshal our Go struct response into JSON and write it to the response writer
	json.NewEncoder(w).Encode(response) // Simpler way to marshal and write JSON

}

// assetsHandler manages creative asset uploads and retrieval
func creativeAssetsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// In a real app, this would list uploaded assets for the user
		fmt.Fprintf(w, "You requested to GET your creative assets (simulated).")
	case http.MethodPost:
		// This will be crucial for handling image/video uploads
		fmt.Fprintf(w, "You requested to POST a new creative asset (simulated upload).")
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
	log.Printf("Received %s request for /assets", r.Method)
}

// modifyImageHandler handles requests to apply GenAI modifications to an image
func modifyImageHandler(w http.ResponseWriter, r *http.Request) {
	// modifyImageHandler handles requests to apply GenAI modifications to an image
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		log.Printf("Error reading modifyImageHandler body: %v", err)
		return
	}

	var req ModifyImageRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		log.Printf("Error unmarshaling modifyImageHandler JSON: %v", err)
		return
	}

	// Simulate AI processing: In a real app, this would call an external AI API
	log.Printf("Simulating AI modification for Image ID: %s with Prompt: '%s'", req.ImageID, req.ModPrompt)

	response := GenericResponse{
		Message: fmt.Sprintf("Image ID '%s' sent for AI modification with prompt '%s' (simulated).", req.ImageID, req.ModPrompt),
		Status:  "processing", // Indicate that processing is ongoing
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted) // Send 202 Accepted status for async processing
	json.NewEncoder(w).Encode(response)
}

func main() {
	// Register our handler functions to specific URL paths
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/users", usersHandler)
	http.HandleFunc("/assets", creativeAssetsHandler)
	http.HandleFunc("/modify-image", modifyImageHandler) // The core AI modification endpoint

	// Define the network address and port for the server to listen on
	// ":" means listen on all available network interfaces. 8080 is a common port for development.
	const addr = ":8080"
	log.Printf("Flowify Server starting on port %s...", addr) // Inform us that the server is starting

	// Start the HTTP server. This function blocks until the server stops or an error occurs.
	// The second argument is an http.Handler. Passing nil means to use the DefaultServeMux,
	// which is where http.HandleFunc registers our handlers.
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		// If there's an error starting the server (e.g., port already in use), log it and exit.
		log.Fatalf("Flowify Server failed to start: %v", err)
	}
}
