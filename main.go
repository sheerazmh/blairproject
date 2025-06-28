package main

import (
	"context"
	"database/sql" // Standard Go SQL package
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver; the underscore means import for side effects (registering itself)

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	// THIS IS THE CORRECT AND RECOMMENDED IMPORT PATH FOR aiplatformpb
	aiplatformpb "cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"google.golang.org/api/option"
	structpb "google.golang.org/protobuf/types/known/structpb" // Keep this import
)

const (
	projectID = "flowifypoc"              // Replace with your actual Google Cloud Project ID
	location  = "australia-southeast1"    // Choose a region where Imagen is available
	modelID   = "imagen-3.0-generate-002" // This is the base model ID for Imagen (e.g., for Imagen on Vertex AI)
)

type UserSignUpRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ModifyImageRequest struct {
	AssetID int    `json:"asset_id"`
	Prompt  string `json:"prompt"`
}

type GenericResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
	AssetID int    `json:"asset_id,omitempty"` // Optional, only for asset-related responses
}

type AIModificationResponse struct {
	ModifiedImageURL string `json:"modified_image_url"`
	Message          string `json:"message"`
	Status           string `json:"status"`
}

// Global variable for the database connection pool
var db *sql.DB

// initDB establishes the database connection
func initDB() {
	connStr := "postgres://sheeraz:brandnew2020@localhost:5432/flowifydb?sslmode=disable"
	var err error
	db, err = sql.Open("pgx", connStr) // Use the "pgx" driver
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}

	// Ping the database to verify the connection
	if err = db.Ping(); err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}

	log.Println("Successfully connected to PostgreSQL database!")

	// Create tables if they don't exist
	createTablesSQL := `
    CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        email TEXT UNIQUE NOT NULL,
        password_hash TEXT NOT NULL,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
    );

    CREATE TABLE IF NOT EXISTS creative_assets (
        id SERIAL PRIMARY KEY,
        user_id INTEGER NOT NULL REFERENCES users(id),
        original_filename TEXT NOT NULL,
        uploaded_path TEXT NOT NULL UNIQUE, -- Path on GCS/local, e.g., "uploads/image.jpg"
        modified_path TEXT, -- Path to modified image
        ai_prompt TEXT,
        status TEXT DEFAULT 'uploaded', -- e.g., 'uploaded', 'processing', 'completed', 'failed'
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
    );
    `
	_, err = db.Exec(createTablesSQL)
	if err != nil {
		log.Fatalf("Error creating tables: %v", err)
	}
	log.Println("Database tables checked/created successfully.")
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the Flowify API! Your creative asset transformation journey begins.")
	log.Println("Received GET request for /")
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		log.Printf("Error reading usersHandler body: %v", err)
		return
	}

	var req UserSignUpRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		log.Printf("Error unmarshaling usersHandler JSON: %v", err)
		return
	}

	// --- Database Insertion ---
	// For now, we'll store the password directly (NOT SECURE FOR PRODUCTION!)
	// In a real app, you would hash the password (e.g., using bcrypt).
	var userID int
	err = db.QueryRow("INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id", req.Email, req.Password).Scan(&userID)
	if err != nil {
		log.Printf("Error inserting new user into DB: %v", err)
		// Check for unique constraint violation (duplicate email)
		if err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"` {
			http.Error(w, "User with this email already exists.", http.StatusConflict) // 409 Conflict
			return
		}
		http.Error(w, "Error registering user", http.StatusInternalServerError)
		return
	}
	log.Printf("User '%s' registered successfully with ID: %d", req.Email, userID)

	response := GenericResponse{
		Message: fmt.Sprintf("User '%s' registered successfully.", req.Email),
		Status:  "success",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201 Created for new resource
	json.NewEncoder(w).Encode(response)
}

func creativeAssetsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		// Ensure the 'uploads' directory exists
		uploadDir := "./uploads"
		if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
			err = os.Mkdir(uploadDir, 0755) // Create directory with read/write/execute permissions for owner, read/execute for others
			if err != nil {
				http.Error(w, "Error creating upload directory: "+err.Error(), http.StatusInternalServerError)
				log.Printf("Error creating upload directory: %v", err)
				return
			}
		}

		// Parse the multipart form data. Max memory 10MB (10 << 20 bytes).
		// Files larger than this will be written to disk in a temp directory.
		err := r.ParseMultipartForm(10 << 20) // 10 MB limit
		if err != nil {
			http.Error(w, "Error parsing multipart form: "+err.Error(), http.StatusInternalServerError)
			log.Printf("Error parsing multipart form for /assets POST: %v", err)
			return
		}

		// Retrieve the file from the form data.
		// "image" is the name of the form field that will contain the file.
		file, handler, err := r.FormFile("image")
		if err != nil {
			http.Error(w, "Error retrieving file from form: "+err.Error(), http.StatusBadRequest)
			log.Printf("Error retrieving 'image' file from form for /assets POST: %v", err)
			return
		}
		// Ensure the uploaded file is closed when the function exits
		defer file.Close()

		// Define the path where the file will be saved locally
		filePath := filepath.Join(uploadDir, handler.Filename)
		dst, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "Error creating file on server: "+err.Error(), http.StatusInternalServerError)
			log.Printf("Error creating destination file '%s': %v", handler.Filename, err)
			return
		}
		// Ensure the destination file is closed when the function exits
		defer dst.Close()

		// Copy the uploaded file's content to the new destination file.
		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, "Error saving file: "+err.Error(), http.StatusInternalServerError)
			log.Printf("Error copying file content to '%s': %v", handler.Filename, err)
			return
		}

		// --- Database Insertion for Creative Asset Metadata ---
		var assetID int
		// For now, hardcode user_id to 1. This will be replaced with real user ID after authentication.
		userID := 1
		assetUploadedPath := filePath // Store the local file path for now (will be GCS URL later)

		// Insert asset metadata into the creative_assets table and retrieve the generated ID
		err = db.QueryRow(
			"INSERT INTO creative_assets (user_id, original_filename, uploaded_path) VALUES ($1, $2, $3) RETURNING id",
			userID, handler.Filename, assetUploadedPath,
		).Scan(&assetID)

		if err != nil {
			http.Error(w, "Error saving asset metadata to database: "+err.Error(), http.StatusInternalServerError)
			log.Printf("Error inserting creative asset metadata into DB: %v", err)
			return
		}

		log.Printf("File '%s' uploaded successfully to local storage at %s. Asset ID: %d", handler.Filename, filePath, assetID)

		// Prepare a success response, including the newly generated AssetID
		response := GenericResponse{
			Message: fmt.Sprintf("File '%s' uploaded successfully. Asset ID: %d", handler.Filename, assetID),
			Status:  "success",
			AssetID: assetID, // Populate the AssetID for the frontend
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // Send 200 OK status
		json.NewEncoder(w).Encode(response)

	case http.MethodGet:
		// Placeholder for getting assets (will come later with database queries and UI listing)
		fmt.Fprintf(w, "You requested to GET your creative assets (list/view assets not implemented yet).")
		log.Printf("Received GET request for /assets")

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// modifyImageHandler handles requests to apply GenAI modifications to an image
func modifyImageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Error reading modifyImageHandler body: %v", err)
		return
	}

	var req ModifyImageRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		http.Error(w, "Invalid JSON format: "+err.Error(), http.StatusBadRequest)
		log.Printf("Error unmarshaling modifyImageHandler JSON: %v", err)
		return
	}

	log.Printf("Calling Google Vertex AI for Asset ID: %d with Prompt: '%s'", req.AssetID, req.Prompt)

	ctx := context.Background()
	client, err := aiplatform.NewPredictionClient(ctx, option.WithEndpoint(fmt.Sprintf("%s-aiplatform.googleapis.com:443", location)))
	if err != nil {
		http.Error(w, "Failed to create Vertex AI client: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Failed to create Vertex AI client: %v", err)
		return
	}
	defer client.Close()

	imageFilename := fmt.Sprintf("%d", req.AssetID)
	imagePath := filepath.Join("./uploads", imageFilename)
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		http.Error(w, "Error reading uploaded image file: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Error reading image file %s: %v", imagePath, err)
		return
	}

	encodedImage := base64.StdEncoding.EncodeToString(imageData)

	// --- THE FINAL, ABSOLUTELY CONFIRMED CORRECTED `instances` PAYLOAD CONSTRUCTION ---
	// This uses structpb.NewStruct to build the inner content, and then correctly
	// wraps it into the aiplatformpb.Value type as expected by the API.

	// 1. Create the struct for the "image" field
	imageStructContent := map[string]interface{}{
		"bytesBase64Encoded": encodedImage,
	}
	imageProtoStruct, err := structpb.NewStruct(imageStructContent)
	if err != nil {
		http.Error(w, "Error creating image proto struct: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Error creating image proto struct: %v", err)
		return
	}

	// 2. Create the struct for the "parameters" field
	parametersStructContent := map[string]interface{}{
		"sampleCount": 1, // Request 1 output image
		// Add other parameters as needed by the model for image manipulation
		// For in-painting/out-painting, you would add mask_image (base64 string) and mask_prompt here.
		// "seed": 123,
		// "mask_image": "base64_mask_image_string",
		// "mask_prompt": "mask this area",
	}
	parametersProtoStruct, err := structpb.NewStruct(parametersStructContent)
	if err != nil {
		http.Error(w, "Error creating parameters proto struct: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Error creating parameters proto struct: %v", err)
		return
	}

	// 3. Create the main instance's struct. This represents the content of one instance.
	// This map also needs to be map[string]any because the values are *structpb.Value types*
	// which implicitly implement the `any` interface.
	mainInstanceStructContent := map[string]interface{}{
		"image":      imageProtoStruct,      // structpb.Struct pointer
		"prompt":     req.Prompt,            // native string
		"parameters": parametersProtoStruct, // structpb.Struct pointer
	}
	mainInstanceProtoStruct, err := structpb.NewStruct(mainInstanceStructContent)
	if err != nil {
		http.Error(w, "Error creating main instance proto struct: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Error creating main instance proto struct: %v", err)
		return
	}

	// 4. Wrap the main instance struct into a structpb.Value.
	// This is the *exact* format required for the Instances field of PredictRequest.
	instanceValue := structpb.NewStructValue(mainInstanceProtoStruct)

	// 5. Create the slice of *structpb.Value instances for the request
	instances := []*structpb.Value{
		instanceValue,
	}
	// --- END FINAL, ABSOLUTELY CONFIRMED CORRECTED `instances` PAYLOAD CONSTRUCTION ---

	endpoint := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s", projectID, location, modelID)

	predictReq := &aiplatformpb.PredictRequest{
		Endpoint:  endpoint,
		Instances: instances, // This now correctly expects []*structpb.Value
	}

	resp, err := client.Predict(ctx, predictReq)
	if err != nil {
		http.Error(w, "Error calling Vertex AI Predict: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Error calling Vertex AI Predict: %v", err)
		return
	}

	if len(resp.GetPredictions()) == 0 {
		http.Error(w, "Vertex AI did not return any predictions.", http.StatusInternalServerError)
		log.Println("Vertex AI response was empty.")
		return
	}

	var modifiedImageURL string
	firstPrediction := resp.GetPredictions()[0]
	if structVal := firstPrediction.GetStructValue(); structVal != nil {
		if bytesBase64Field := structVal.Fields["bytesBase64Encoded"]; bytesBase64Field != nil {
			if base64Str := bytesBase64Field.GetStringValue(); base64Str != "" {
				decodedImage, decodeErr := base64.StdEncoding.DecodeString(base64Str)
				if decodeErr != nil {
					http.Error(w, "Error decoding image from AI response: "+decodeErr.Error(), http.StatusInternalServerError)
					log.Printf("Error decoding base64 image from AI: %v", decodeErr)
					return
				}
				modifiedFilename := fmt.Sprintf("modified_%d", req.AssetID)
				modifiedImagePath := filepath.Join("./uploads", modifiedFilename)
				if writeErr := os.WriteFile(modifiedImagePath, decodedImage, 0644); writeErr != nil {
					http.Error(w, "Error saving modified image: "+writeErr.Error(), http.StatusInternalServerError)
					log.Printf("Error saving modified image to %s: %v", modifiedImagePath, writeErr)
					return
				}
				modifiedImageURL = fmt.Sprintf("/uploads/%s", modifiedFilename)
				modifiedImageURL = fmt.Sprintf("/uploads/%s", modifiedFilename)
			}
		}
	}

	if modifiedImageURL == "" {
		http.Error(w, "Failed to extract modified image URL from AI response.", http.StatusInternalServerError)
		log.Printf("AI response structure unexpected: %+v", firstPrediction)
		return
	}

	log.Printf("Vertex AI prediction complete. Modified Image saved and URL: %s", modifiedImageURL)
	response := AIModificationResponse{
		ModifiedImageURL: modifiedImageURL,
		Message:          fmt.Sprintf("AI modification complete via Google Vertex AI for asset ID '%d'. Prompt: '%s'", req.AssetID, req.Prompt),
		Status:           "success",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	log.Println("AI modification response sent successfully.")
}

func main() {

	initDB() // Initialize the database connection
	mux := http.NewServeMux()

	mux.HandleFunc("/users", usersHandler)
	mux.HandleFunc("/assets", creativeAssetsHandler)
	mux.HandleFunc("/modify-image", modifyImageHandler)

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/static/index.html", http.StatusFound)
	})

	const addr = ":8080"
	log.Printf("Flowify Server starting on port %s...", addr)

	err := http.ListenAndServe(addr, mux)
	if err != nil {
		log.Fatalf("Flowify Server failed to start: %v", err)
	}
}
