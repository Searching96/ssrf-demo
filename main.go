package main

import (
	"fmt"
	"io"
	"net/http"
)

// The Vulnerable Service
func fetchProfilePicture(w http.ResponseWriter, r *http.Request) {
	imageURL := r.URL.Query().Get("url")

	// DANGER: Fetching without validation
	resp, err := http.Get(imageURL)
	if err != nil {
		http.Error(w, "Failed to fetch image", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Pass the content type (image, text, json) back to the browser
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	io.Copy(w, resp.Body)
}

func main() {
	// 1. Internal Secret API (Victim 1)
	http.HandleFunc("/internal/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status": "healthy", "active_users": 420, "db_password": "super_secret_db_pass"}`)
	})

	// 2. The Vulnerable Proxy Route
	http.HandleFunc("/api/v1/import-avatar", fetchProfilePicture)

	// 3. A Simple Visual Frontend for your Presentation
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `
			<html>
			<body style="font-family: sans-serif; padding: 2rem;">
				<h2>Avatar Importer (Vulnerable)</h2>
				<p>Enter an image URL to import your avatar:</p>
				
				<!-- This form triggers the SSRF -->
				<form action="/api/v1/import-avatar" method="GET">
					<input type="text" name="url" value="https://github.com/github.png" style="width: 400px; padding: 5px;">
					<button type="submit" style="padding: 5px;">Fetch Avatar</button>
				</form>
			</body>
			</html>
		`)
	})

	fmt.Println("Server running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
