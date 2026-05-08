package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func fetchProfilePicture(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL.Query().Get("url")

	resp, err := http.Get(targetURL)
	if err != nil {
		// If it fails, print the EXACT system error to the screen
		http.Error(w, "Proxy Failed. System Error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	io.Copy(w, resp.Body)
}

func main() {
	// Dynamically grab the port Render is using
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 1. Internal Metrics Mock
	http.HandleFunc("/internal/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status": "UP", "DB_PASSWORD": "super_secret_production_password"}`)
	})

	// 2. AWS Metadata Mock
	http.HandleFunc("/latest/meta-data/iam/security-credentials/WAF-Admin-Role", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"AccessKeyId": "AKIA_STOLEN_CAPITAL_ONE_KEYS", "SecretAccessKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"}`)
	})

	// 3. Frontend with DYNAMIC Ports and UI Updates
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		// We inject the actual running port directly into the HTML so it never fails
		html := fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>Avatar Importer</title>
			</head>
			<body style="font-family:sans-serif; padding:2rem;">
				<h2>Corporate Avatar Importer</h2>
				
				<form action="/proxy" method="GET" style="max-width: 400px;">
					<label style="display:block; margin-bottom:5px; font-weight:bold;">Import Profile Image URL:</label>
					
					<input type="text" name="url" value="https://via.placeholder.com/150" 
					       style="width: 100%%; padding: 10px; margin-bottom: 10px; box-sizing: border-box; border: 1px solid #ccc; border-radius: 4px;">
					       
					<button type="submit" 
					        style="width: 100%%; padding: 10px; cursor: pointer; box-sizing: border-box; background: #007BFF; color: white; border: none; border-radius: 4px; font-weight: bold;">
					        Fetch
					</button>
				</form>
				
				<hr style="margin: 30px 0;">
				
				<div style="background: #f8d7da; padding: 15px; border-radius: 5px; border: 1px solid #f5c6cb; max-width: 800px;">
					<h3 style="color: #721c24; margin-top: 0;">Presentation Demo Links (Click to Exploit)</h3>
					
					<p><b>Stage 1:</b><br>
					<a href="/proxy?url=https://via.placeholder.com/150">Fetch External Image</a></p>
					
					<p><b>Stage 2 (Internal Leak):</b><br>
					<a href="/proxy?url=http://127.0.0.1:%s/internal/metrics">Steal Internal DB Passwords</a></p>
					
					<p><b>Stage 3 (Cloud Breach):</b><br>
					<a href="/proxy?url=http://127.0.0.1:%s/latest/meta-data/iam/security-credentials/WAF-Admin-Role">Steal AWS Metadata Keys</a></p>
				</div>
			</body>
			</html>
		`, port, port) // Injects the port variable into the %s placeholders

		fmt.Fprint(w, html)
	})

	http.HandleFunc("/proxy", fetchProfilePicture)

	fmt.Println("Server running on port " + port)
	http.ListenAndServe(":"+port, nil)
}
