package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"      // For logging server messages to the console
	"net/http" // The core package for building web servers
	"os"
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
// creativeAssetsHandler manages creative asset uploads and retrieval
func creativeAssetsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Placeholder for getting assets (will come later with database/storage)
		fmt.Fprintf(w, "You requested to GET your creative assets (simulated).")
		log.Printf("Received GET request for /assets")
	case http.MethodPost:
		// --- FILE UPLOAD LOGIC ---
		// 1. Parse the multipart form data. Max memory 10MB (10 << 20 bytes).
		// Files larger than this will be written to disk in a temp directory.
		err := r.ParseMultipartForm(10 << 20) // 10 MB limit
		if err != nil {
			http.Error(w, "Error parsing multipart form: "+err.Error(), http.StatusInternalServerError)
			log.Printf("Error parsing multipart form for /assets POST: %v", err)
			return
		}

		// 2. Retrieve the file from the form data.
		// "image" is the name of the form field that will contain the file.
		// When building the frontend, your <input type="file" name="image">
		// will need to match this "image" key.
		file, handler, err := r.FormFile("image")
		if err != nil {
			http.Error(w, "Error retrieving file from form: "+err.Error(), http.StatusBadRequest)
			log.Printf("Error retrieving 'image' file from form for /assets POST: %v", err)
			return
		}
		// Ensure the uploaded file is closed when the function exits
		defer file.Close()

		// 3. Create a destination file on the server to save the uploaded content.
		// For now, we'll save it directly in the 'blairproject' directory.
		// In a real app, you'd save it to a dedicated 'uploads' folder or cloud storage.
		dst, err := os.Create(handler.Filename) // Use the original filename
		if err != nil {
			http.Error(w, "Error creating file on server: "+err.Error(), http.StatusInternalServerError)
			log.Printf("Error creating destination file '%s': %v", handler.Filename, err)
			return
		}
		// Ensure the destination file is closed when the function exits
		defer dst.Close()

		// 4. Copy the uploaded file's content to the new destination file.
		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, "Error saving file: "+err.Error(), http.StatusInternalServerError)
			log.Printf("Error copying file content to '%s': %v", handler.Filename, err)
			return
		}

		// 5. Send a success response
		response := GenericResponse{
			Message: fmt.Sprintf("File '%s' uploaded successfully with size %d bytes (simulated).", handler.Filename, handler.Size),
			Status:  "success",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)

		log.Printf("File '%s' uploaded successfully to local storage.", handler.Filename)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
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
