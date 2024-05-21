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
	"Hi, Astral Expressers... Well, you caught me.",
	"All space and time are practically infinite, and yet right here, right now, we find ourselves together. That's the nature of 'destiny' — it creates a miracle but convinces you of an accident.",
	"Oh, bye-bye, (Trailblazer). See if you can surprise me next time.",
	"Before I joined the Stellaron Hunters, the nature of my work meant that I barely ever saw the same person twice.",
	"Elio said that I'm good at creating 'fear,' even though I don't know what it is.",
	"Playing the violin and firing a gun both require flexible fingers, but bullets are more obedient.",
	"I especially love velvet coats, they're so fragile and beautiful. Difficult to maintain — you only have to be a tiny bit careless to ruin the sheen.",
	"The past and the future are so similar to each other. I'm indifferent towards them.",
	"There's a planet I go to every summer to look at the sea. That's when the tides are fiercest — you have to stand far away on the shore. Then, one year, they constructed a long observation pier. I haven't been back since.",
	"My home world is one of many planets changed by a Stellaron... *sigh* It's a shame I never got to witness how far it fell at the time.",
	"I like chatting with Silver Wolf. She's got a lot of ideas for someone so small.",
	"Bladie... He takes after his name — his fights are a pleasure to witness.",
	"Oh, Sam isn't nearly as picky about his prey as I am... you might consider it a lucky break running into me.",
	"No one can completely grasp another's thoughts... Not even you or I.",
	"If destiny can't propel me forward, I'll be the one to push destiny.",
	"It's tough to get going again once you've stopped, don't you think?",
	"The path to the future begins right here.",
	"(Trailblazer), we meet again.",
	"Let's skip the formalities, Silver Wolf. I'm always game for putting on a show.",
	"The hunt beckons, Bladie. Are you ready?",
	"Oh, the Express Crew. Looks like we'll be joining forces quite often, huh?",
	"Caught in the net.",
	"Just in time.",
	"May as well kill them all~",
	"That breathing sensation? Remember it.",
	"Time to move.",
	"This won't take long.",
	"Didn't hurt.",
	"Not bad.",
	"Good times never last.",
	"Time to say bye. BOOM.",
	"Relax.",
	"Stand still.",
	"This... isn't the end.",
	"Oh! I'm still alive.",
	"Thanks. You're too good to me.",
	"Does that hurt?",
	"The human body is beautiful in its fragility.",
	"Hmm. We can use it.",
	"No surprise there.",
	"Like what you see? Me too.",
	"Nice work.",
	"Impressive. Want a prize?",
	"Huh, they don't look happy to see us...",
	"*sigh* Is that a stain? I like this coat.",
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
		port = "14467"
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
