package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// --- 1. THE VULNERABLE SERVICE (The "Proxy") ---
func fetchProfilePicture(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL.Query().Get("url")

	// THE VULNERABILITY: Blind trust in user-provided URL
	resp, err := http.Get(targetURL)
	if err != nil {
		http.Error(w, "Error fetching resource", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	io.Copy(w, resp.Body)
}

func main() {
	// --- 2. VERBOSE INTERNAL API (Local Attack Vector) ---
	// This simulates an internal-only admin or dev-ops tool
	http.HandleFunc("/internal/metrics", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]interface{}{
			"status": "UP",
			"uptime": time.Since(time.Now()).String(),
			"environment_variables": map[string]string{
				"DB_PASSWORD": "p@ssword123_production",
				"REDIS_URL":   "redis://internal-cache.local:6379",
				"DEBUG_MODE":  "true",
			},
			"internal_network": []string{"10.0.0.5:db", "10.0.0.8:auth", "10.0.0.12:logs"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	})

	// --- 3. MOCK AWS METADATA (Cloud Attack Vector) ---
	http.HandleFunc("/latest/meta-data/iam/security-credentials/WAF-Admin-Role", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"AccessKeyId": "AKIA_STOLEN_CAPITAL_ONE_KEYS", "SecretAccessKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", "Token": "IQoJb3JpZ2lu..."}`)
	})

	// --- 4. MOCK S3 (The Destination) ---
	http.HandleFunc("/s3/customer-backups-2026/social_security_numbers.csv", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			http.Error(w, "Access Denied: 403 Forbidden", http.StatusForbidden)
			return
		}
		fmt.Fprint(w, "Name, SSN, Balance\nAlice, 000-11-2222, $5M\nBob, 999-88-7777, $150")
	})

	// --- 5. FRONTEND & PROXY ROUTE ---
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `
			<body style="font-family:sans-serif; padding:2rem; line-height:1.6;">
				<h1>My-App Dashboard</h1>
				<form action="/proxy" method="GET">
					<label>Import Profile Image URL:</label><br>
					<input type="text" name="url" value="https://via.placeholder.com/150" style="width:400px; padding:8px;">
					<button type="submit">Fetch</button>
				</form>
				<hr>
				<h3>Demo Commands:</h3>
				<ul>
					<li><b>Internal API:</b> <code>http://localhost:8080/internal/metrics</code></li>
					<li><b>AWS Keys:</b> <code>http://localhost:8080/latest/meta-data/iam/security-credentials/WAF-Admin-Role</code></li>
				</ul>
			</body>
		`)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default for local testing
	}

	fmt.Printf("Server running on port %s\n", port)
	http.ListenAndServe(":"+port, nil)
}
