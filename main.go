package main

import (
	"fmt"
	"log"      // For logging server messages to the console
	"net/http" // The core package for building web servers
)

// homeHandler serves the root path "/"
func homeHandler(w http.ResponseWriter, r *http.Request) {
	// fmt.Fprintf writes formatted output to the ResponseWriter (w)
	fmt.Fprintf(w, "Welcome to the Flowify API! Your creative asset transformation journey begins.")
	log.Println("Received GET request for /") // Log the request for debugging
}

// usersHandler manages user-related actions (signup, login, profile)
func usersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// In a real app, this might list users (admin only) or show a profile
		fmt.Fprintf(w, "You requested to GET user information (simulated).")
	case http.MethodPost:
		// This would handle user registration or login
		fmt.Fprintf(w, "You requested to POST a new user (simulated signup/login).")
	default:
		// For any other method (PUT, DELETE, etc.), send a 405 Method Not Allowed status
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
	log.Printf("Received %s request for /users", r.Method)
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
	if r.Method != http.MethodPost { // GenAI modifications typically require sending data, so POST is appropriate
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	// In a real app, this would process the incoming image and prompt, then call the GenAI model
	fmt.Fprintf(w, "You submitted an image for AI modification with a prompt (simulated).")
	log.Println("Received POST request for /modify-image")
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
