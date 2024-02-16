package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	_ "github.com/lib/pq"
)

var db *sql.DB

func init() {
	var err error
	db, err = InitDB()
	if err != nil {
		panic(err)
	}
}

func main() {
	defer db.Close()

	http.HandleFunc("/", WelcomeHandler)          // Serve the welcome page
	http.HandleFunc("/register", RegisterHandler) // New registration endpoint
	http.HandleFunc("/login", LoginHandler)
	http.HandleFunc("/profile", MainHandler)
	http.HandleFunc("/compile", compileAndSubmitHandler)

	fmt.Println("Server is running on :8080...")
	http.ListenAndServe(":8080", nil)
}

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "123"
	dbname   = "syrma20"
)

func InitDB() (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	fmt.Println("Successfully connected to the database")

	// Create the "users" table if it doesn't exist
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		name VARCHAR(50) NOT NULL,
		surname VARCHAR(50) NOT NULL,
		barcode INTEGER UNIQUE NOT NULL,
		email VARCHAR(50) UNIQUE NOT NULL,
		password VARCHAR(50) NOT NULL
	);
	
	
	`
	_, err = db.Exec(createTableQuery)
	if err != nil {
		return nil, err
	}

	fmt.Println("Users table created successfully")

	// Create the "submitted_code" table if it doesn't exist
	_, err = db.Exec(createCodeTableQuery)
	if err != nil {
		return nil, err
	}

	fmt.Println("Submitted code table created successfully")

	return db, nil
}

func WelcomeHandler(w http.ResponseWriter, r *http.Request) {
	// w.Header().Set("Content-Type", "text/html")
	// http.ServeFile(w, r, "welcome.html")
	http.ServeFile(w, r, "salem.html")

}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		name := r.FormValue("name")
		surname := r.FormValue("surname")
		barcode := r.FormValue("barcode")
		email := r.FormValue("email")
		password := r.FormValue("password")

		// Check if the name already exists in the database
		if exists, err := checkUserExists(name); err != nil {
			fmt.Fprintf(w, "Error checking name: %v", err)
			return
		} else if exists {
			fmt.Fprintf(w, "name already exists. Please choose another name.")
			return
		}

		// Create a new user in the database
		if err := createUser(name, surname, barcode, email, password); err != nil {
			fmt.Fprintf(w, "Error creating user: %v", err)
			return
		}

		// Redirect to the login page after successful registration
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Render the registration page
	http.ServeFile(w, r, "index.html")
}

func checkUserExists(name string) (bool, error) {
	query := "SELECT COUNT(*) FROM users WHERE name = $1"
	var count int
	err := db.QueryRow(query, name).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func createUser(name, surname, barcode, email, password string) error {
	barcodeInt, err := strconv.Atoi(barcode)
	if err != nil {
		return err
	}
	query := "INSERT INTO users (name, surname, barcode, email, password) VALUES ($1, $2, $3, $4, $5)"
	_, err = db.Exec(query, name, surname, barcodeInt, email, password)
	return err
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	// w.Header().Set("Content-Type", "text/html")

	// // Serve the login page initially
	// fmt.Println("Serving login page from:", "login.html")
	// http.ServeFile(w, r, "login.html")

	// Check if the request method is POST
	if r.Method == http.MethodPost {
		name := r.FormValue("name")
		password := r.FormValue("password")

		// Validate credentials against the database
		if isValid, err := validateCredentials(name, password); err == nil && isValid {
			// Set session/token (you might want to use a proper session management library)
			// For simplicity, we'll just set a cookie for demonstration purposes
			http.SetCookie(w, &http.Cookie{
				Name:  "session",
				Value: "your_session_token",
				Path:  "/",
			})

			// Redirect to the main page after successful login
			http.Redirect(w, r, "/profile", http.StatusSeeOther)
			return
		} else {
			// Credentials are incorrect or an error occurred, show an error message
			fmt.Fprintf(w, "Invalid credentials. Please try again.")
			return
		}

	}
	//Render the registration page
	http.ServeFile(w, r, "login.html")
}

func MainHandler(w http.ResponseWriter, r *http.Request) {
	// Check if the user is authenticated (you might want to use a proper session management library)
	cookie, err := r.Cookie("session")
	if err != nil || cookie.Value != "your_session_token" {
		// User is not authenticated, redirect to the login page
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// User is authenticated, render the main page
	http.ServeFile(w, r, "profile.html")
}

func validateCredentials(name, password string) (bool, error) {
	query := "SELECT COUNT(*) FROM users WHERE name = $1 AND password = $2"
	var count int
	err := db.QueryRow(query, name, password).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Define a new table to store submitted code
const createCodeTableQuery = `
CREATE TABLE IF NOT EXISTS submitted_code (
    id SERIAL PRIMARY KEY,
    code TEXT NOT NULL
);
`

// Modify compileAndSubmitHandler to save the submitted code to the database
func compileAndSubmitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// Read the code from the request body
		code, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		// Insert the code into the database
		_, err = db.Exec("INSERT INTO submitted_code (code) VALUES ($1)", string(code))
		if err != nil {
			http.Error(w, "Failed to insert code into database", http.StatusInternalServerError)
			return
		}

		// Return success message
		fmt.Fprintf(w, "Code submitted successfully\n")
	} else if r.Method == http.MethodGet {
		// Serve the compile.html file for GET requests
		http.ServeFile(w, r, "compile.html")
	}
}
