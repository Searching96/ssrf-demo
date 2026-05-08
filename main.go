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

	// 1. Internal Metrics Mock (Verbose)
	http.HandleFunc("/internal/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"application": "avatar-importer-service",
			"status": "UP",
			"uptime": "14d 5h 23m",
			"system": {
				"cpu_usage_percent": 42.5,
				"memory_alloc_mb": 1024,
				"goroutines": 156
			},
			"environment_variables": {
				"NODE_ENV": "production",
				"SERVER_PORT": "8080",
				"DB_HOST": "10.0.4.55",
				"DB_PORT": "5432",
				"DB_USER": "admin_prod",
				"DB_PASSWORD": "super_secret_production_password",
				"REDIS_CACHE_URL": "redis://:cache_pass_9982@internal-redis.local:6379",
				"PAYMENT_API_KEY": "sk_live_51HXYZ8B..."
			},
			"internal_network_map": [
				"10.0.1.10 (API Gateway)",
				"10.0.4.55 (Primary Postgres DB)",
				"10.0.5.12 (Internal Payment Processor)"
			]
		}`)
	})

	// 2. AWS Metadata Mock (Verbose & Authentic Schema)
	http.HandleFunc("/latest/meta-data/iam/security-credentials/WAF-Admin-Role", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Note: Using "ASIA" instead of "AKIA" because AWS temporary STS tokens always start with ASIA.
		// This detail shows you really know your cloud security!
		fmt.Fprint(w, `{
			"Code": "Success",
			"LastUpdated": "2026-05-09T02:35:14Z",
			"Type": "AWS-HMAC",
			"AccessKeyId": "AWS_ASIA_STOLEN_CAPITAL_ONE_KEYS",
			"SecretAccessKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			"Token": "IQoJb3JpZ2luX2VjEAIaCXVzLWVhc3QtMSJGMEQCIGD1p/3lxyz98ABCDEFghIJKlmnopQRSTuvWXYZ12345AiB67890abcdefGHIjklmNOPQrsTUVwxyz1234567890KlwEI3P//////////8BEAMaDDEyMzQ1Njc4OTAxMiIMABCDEFGHijklMNOPQRstUVwxYZ1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890==",
			"Expiration": "2026-05-09T08:35:14Z"
		}`)
	})

	// 3. Mock private S3 bucket
	http.HandleFunc("/s3/customer-backups-2026/social_security_numbers.csv", func(w http.ResponseWriter, r *http.Request) {
		// We check for the specific stolen keys in the Authorization header
		authHeader := r.Header.Get("Authorization")
		expectedAuth := "AWS_ASIA_STOLEN_CAPITAL_ONE_KEYS"

		if authHeader != expectedAuth {
			// If no keys are provided OR they are incorrect, mimic AWS Access Denied
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, "<?xml version=\"1.0\" encoding=\"UTF-8\"?><Error><Code>AccessDenied</Code><Message>Access Denied</Message></Error>")
			return
		}

		// If the EXACT stolen keys ARE provided, release the private data
		w.Header().Set("Content-Type", "text/csv")
		fmt.Fprint(w, "Name,SSN,AccountBalance\nAlice,000-11-2222,$5000000\nBob,999-88-7777,$150\nCharlie,888-55-4444,$250000")
	})

	// 4. Frontend with DYNAMIC Ports and UI Updates
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
					
					<input type="text" name="url" value="https://i.pinimg.com/1200x/d6/c4/ce/d6c4ce17fe18176c7d433021079fe1aa.jpg" 
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
					<a href="/proxy?url=https://i.pinimg.com/1200x/d6/c4/ce/d6c4ce17fe18176c7d433021079fe1aa.jpg">Fetch External Image</a></p>
					
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
