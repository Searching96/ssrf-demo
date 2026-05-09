package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

// --- VULNERABLE HANDLER ---
func fetchProfilePicture(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL.Query().Get("url")

	resp, err := http.Get(targetURL)
	if err != nil {
		http.Error(w, "Proxy Failed. System Error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	io.Copy(w, resp.Body)
}

// --- SECURE HANDLER: NETWORK DENYLISTING & DNS REBINDING PREVENTION ---
func fetchProfilePictureFixed(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL.Query().Get("url")

	// 1. Parse the URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		http.Error(w, "Error: Invalid URL format.", http.StatusBadRequest)
		return
	}

	// 2. Block Obfuscated Decimal IPs
	// Valid top-level domains (.com, .net) cannot be entirely numeric.
	// If this parses perfectly as an integer, it is an obfuscated IP bypass attempt.
	hostname := parsedURL.Hostname()
	if _, err := strconv.ParseInt(hostname, 10, 64); err == nil {
		errorMessage := fmt.Sprintf("SSRF Blocked: Obfuscated decimal IP detected (%s).", hostname)
		http.Error(w, errorMessage, http.StatusForbidden)
		return
	}

	// 3. Resolve the Domain to an IP Address (Time of Check)
	ips, err := net.LookupIP(hostname)
	if err != nil || len(ips) == 0 {
		http.Error(w, "Error: Could not resolve hostname.", http.StatusBadRequest)
		return
	}

	// We take the first resolved IP for our check and our connection
	resolvedIP := ips[0]

	// 4. The Denylist Check
	// IsLoopback blocks 127.0.0.1 (Localhost)
	// IsPrivate blocks 10.x.x.x, 172.16.x.x, 192.168.x.x (Internal Networks)
	// IsLocalUnicast blocks 169.254.x.x (Link-Local, including AWS Metadata)
	if resolvedIP.IsLoopback() || resolvedIP.IsPrivate() || resolvedIP.IsLinkLocalUnicast() {
		errorMessage := fmt.Sprintf("SSRF Blocked: The IP address (%s) belongs to a restricted internal network.", resolvedIP.String())
		http.Error(w, errorMessage, http.StatusForbidden)
		return
	}

	// 5. PREVENT DNS REBINDING: Create a custom HTTP Transport
	// We force the network dialer to use the exact IP we just validated,
	// completely ignoring any subsequent DNS lookups. (Time of Use)
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Extract the port from the original request URL
			port := parsedURL.Port()
			if port == "" {
				if parsedURL.Scheme == "https" {
					port = "443"
				} else {
					port = "80"
				}
			}

			// Construct the forced address using our validated IP
			safeAddr := net.JoinHostPort(resolvedIP.String(), port)

			// Dial using the safe IP directly
			dialer := &net.Dialer{}
			return dialer.DialContext(ctx, network, safeAddr)
		},
	}

	// 6. Create a custom HTTP client using our hardened transport
	client := &http.Client{
		Transport: transport,
	}

	// 7. Execute the request safely
	req, err := http.NewRequestWithContext(r.Context(), "GET", targetURL, nil)
	if err != nil {
		http.Error(w, "Error building request.", http.StatusInternalServerError)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Proxy Failed.", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	io.Copy(w, resp.Body)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Mock Endpoints for the Demo
	// 1. Internal Metrics Mock
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

	// 3. Private S3 Bucket Mock
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

	// Frontend UI
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		html := fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
			<head><title>Avatar Importer</title></head>
			<body style="font-family:sans-serif; padding:2rem;">
				<h2>Corporate Avatar Importer</h2>
				
				<!-- VULNERABLE FORM -->
				<div style="background: #f8d7da; padding: 15px; border: 1px solid #f5c6cb; border-radius: 5px; max-width: 400px; margin-bottom: 20px;">
					<h3 style="color: #721c24; margin-top: 0;">Vulnerable Importer (v1.0)</h3>
					<form action="/api/v1/avatar/import" method="GET">
						<input type="text" name="url" value="https://i.pinimg.com/1200x/d6/c4/ce/d6c4ce17fe18176c7d433021079fe1aa.jpg" style="width: 100%%; padding: 10px; margin-bottom: 10px; box-sizing: border-box;">
						<button type="submit" style="width: 100%%; padding: 10px; cursor: pointer; background: #dc3545; color: white; border: none; font-weight: bold; border-radius: 4px;">Exploit Fetch</button>
					</form>
				</div>

				<!-- SECURE FORM -->
				<div style="background: #d4edda; padding: 15px; border: 1px solid #c3e6cb; border-radius: 5px; max-width: 400px;">
					<h3 style="color: #155724; margin-top: 0;">Secure Importer (v2.0)</h3>
					<form action="/api/v2/avatar/import" method="GET">
						<input type="text" name="url" value="https://i.pinimg.com/1200x/d6/c4/ce/d6c4ce17fe18176c7d433021079fe1aa.jpg" style="width: 100%%; padding: 10px; margin-bottom: 10px; box-sizing: border-box;">
						<button type="submit" style="width: 100%%; padding: 10px; cursor: pointer; background: #28a745; color: white; border: none; font-weight: bold; border-radius: 4px;">Secure Fetch</button>
					</form>
				</div>
				
				<hr style="margin: 30px 0; max-width: 800px;">
				
				<div style="background: #e2e3e5; padding: 15px; border-radius: 5px; max-width: 800px;">
					<h3 style="margin-top: 0;">Presentation Exploit Links</h3>
					<p><a href="/api/v1/avatar/import?url=http://127.0.0.1:%s/internal/metrics">1. Attack Internal Network (Vulnerable)</a></p>
					<p><a href="/api/v2/avatar/import?url=http://127.0.0.1:%s/internal/metrics">2. Attack Internal Network (Secure)</a></p>
					<p><a href="/api/v1/avatar/import?url=http://127.0.0.1:%s/latest/meta-data/iam/security-credentials/WAF-Admin-Role">3. Attack Cloud Metadata (Vulnerable)</a></p>
					<p><a href="/api/v2/avatar/import?url=http://127.0.0.1:%s/latest/meta-data/iam/security-credentials/WAF-Admin-Role">4. Attack Cloud Metadata (Secure)</a></p>
				</div>
			</body>
			</html>
		`, port, port, port, port)
		fmt.Fprint(w, html)
	})

	// Register Routes
	http.HandleFunc("/api/v1/avatar/import", fetchProfilePicture)
	http.HandleFunc("/api/v2/avatar/import", fetchProfilePictureFixed)

	fmt.Println("Server running on port " + port)
	http.ListenAndServe(":"+port, nil)
}
