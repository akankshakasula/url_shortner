package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

var urlMappings = make(map[string]string)

var mapLocker sync.RWMutex

const shortCodeLength = 6

const codeCharacters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func init() {
	rand.Seed(time.Now().UnixNano())
}

func generateUniqueShortCode() string {
	for {
		shortCodeBytes := make([]byte, shortCodeLength)
		for i := range shortCodeBytes {
			randomIndex := rand.Intn(len(codeCharacters))
			shortCodeBytes[i] = codeCharacters[randomIndex]
		}
		newCode := string(shortCodeBytes)

		mapLocker.RLock()
		_, alreadyExists := urlMappings[newCode]
		mapLocker.RUnlock()

		if !alreadyExists {
			return newCode
		}
	}
}

func handleShortenRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Please use POST method to shorten URLs.", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read your request.", http.StatusInternalServerError)
		return
	}

	type URLRequest struct {
		LongURL string `json:"url"`
	}
	var incomingURL URLRequest
	if err := json.Unmarshal(bodyBytes, &incomingURL); err != nil {
		http.Error(w, "Invalid JSON. Make sure you send {'url': 'your_long_url'}.", http.StatusBadRequest)
		return
	}

	if incomingURL.LongURL == "" {
		http.Error(w, "URL cannot be empty.", http.StatusBadRequest)
		return
	}
	if !strings.HasPrefix(incomingURL.LongURL, "http://") && !strings.HasPrefix(incomingURL.LongURL, "https://") {
		http.Error(w, "URL must start with http:// or https://.", http.StatusBadRequest)
		return
	}

	newShortCode := generateUniqueShortCode()

	mapLocker.Lock()
	urlMappings[newShortCode] = incomingURL.LongURL
	mapLocker.Unlock()

	type ShortenResponse struct {
		ShortenedLink string `json:"short_url"`
	}
	response := ShortenResponse{
		ShortenedLink: fmt.Sprintf("http://localhost:8080/%s", newShortCode),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleRedirectRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Please use GET method to access short URLs.", http.StatusMethodNotAllowed)
		return
	}

	shortCode := strings.TrimPrefix(r.URL.Path, "/")
	if shortCode == "" {
		http.Error(w, "Short code not found in URL.", http.StatusBadRequest)
		return
	}

	mapLocker.RLock()
	longURL, found := urlMappings[shortCode]
	mapLocker.RUnlock()

	if !found {
		http.Error(w, "Short URL not found in our records.", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, longURL, http.StatusFound)
}

func main() {
	log.SetOutput(io.Discard) // Set log output to discard all messages

	fmt.Println("Starting our simple Go URL Shortener!")

	http.HandleFunc("/shorten", handleShortenRequest)

	http.HandleFunc("/", handleRedirectRequest)

	listenPort := ":8080"
	fmt.Printf("Server listening on http://localhost%s\n", listenPort)
	log.Fatal(http.ListenAndServe(listenPort, nil))
}
