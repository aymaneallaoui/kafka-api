package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type Response struct {
	Message string `json:"message"`
	URL     string `json:"url,omitempty"`
}

var quotes = []string{
	"Quote 1 from Kafka.",
	"Quote 2 from Kafka.",
	"Quote 3 from Kafka.",
}

var imageFiles []string
var quoteIndex int
var imageIndex int
var mu sync.Mutex

func main() {
	var err error
	imageDir := os.Getenv("IMAGE_DIR")
	if imageDir == "" {
		imageDir = "./images"
	}
	imageFiles, err = loadImages(imageDir)
	if err != nil {
		log.Fatalf("failed to load images: %v", err)
	}

	shuffleQuotes()
	shuffleImages()

	go shufflePeriodically()

	mux := http.NewServeMux()
	mux.HandleFunc("/quote", quoteHandler)
	mux.HandleFunc("/image", imageHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	log.Println("Server is running on port " + port + "...")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}

func loadImages(dir string) ([]string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var imageFiles []string
	for _, file := range files {
		if !file.IsDir() {
			imageFiles = append(imageFiles, file.Name())
		}
	}
	return imageFiles, nil
}

func shuffleQuotes() {
	mu.Lock()
	defer mu.Unlock()
	shuffle(len(quotes), func(i, j int) { quotes[i], quotes[j] = quotes[j], quotes[i] })
	quoteIndex = 0
}

func shuffleImages() {
	mu.Lock()
	defer mu.Unlock()
	shuffle(len(imageFiles), func(i, j int) { imageFiles[i], imageFiles[j] = imageFiles[j], imageFiles[i] })
	imageIndex = 0
}

func shuffle(n int, swap func(i, j int)) {
	for i := n - 1; i > 0; i-- {
		j, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		swap(i, int(j.Int64()))
	}
}

func shufflePeriodically() {
	for {
		time.Sleep(1 * time.Hour)
		shuffleQuotes()
		shuffleImages()
	}
}

func quoteHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	quote := quotes[quoteIndex]
	quoteIndex = (quoteIndex + 1) % len(quotes)
	mu.Unlock()

	response := Response{
		Message: quote,
	}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	imageFile := imageFiles[imageIndex]
	imageIndex = (imageIndex + 1) % len(imageFiles)
	mu.Unlock()

	imagePath := filepath.Join("./images", imageFile)
	http.ServeFile(w, r, imagePath)
}
