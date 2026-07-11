package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Snowy-Girl-Labs/codepom/gateway/pkg/queue"
)

type Server struct {
	q queue.QueueProvider
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	// Initialize the memory queue provider
	qProvider := queue.NewMemoryProvider()
	srv := &Server{q: qProvider}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/webhooks/sonarqube", srv.handleSonarQubeWebhook)
	mux.HandleFunc("/webhooks/github", srv.handleGitHubWebhook)

	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Start queue worker daemon
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go srv.workerLoop(ctx)

	// Graceful shutdown listener
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("🐾 CodePom Gateway starting on port %s...", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Gateway server failed: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down gateway gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}
	log.Println("Gateway stopped.")
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleSonarQubeWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Enqueue the triage job
	jobID, err := s.q.Enqueue(r.Context(), "sonarqube_triage", body)
	if err != nil {
		log.Printf("Failed to enqueue job: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("🐾 Enqueued SonarQube triage job: %s", jobID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{"jobId": jobID, "status": "queued"})
}

func (s *Server) handleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	jobID, err := s.q.Enqueue(r.Context(), "github_review", body)
	if err != nil {
		log.Printf("Failed to enqueue job: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("🐾 Enqueued GitHub review job: %s", jobID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{"jobId": jobID, "status": "queued"})
}

// workerLoop polls the queue sequentially and runs worker tasks
func (s *Server) workerLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	log.Println("🐾 Background Queue Worker started.")

	for {
		select {
		case <-ctx.Done():
			log.Println("Queue Worker stopping...")
			return
		case <-ticker.C:
			job, err := s.q.Dequeue(ctx)
			if err != nil {
				log.Printf("Worker error dequeuing job: %v", err)
				continue
			}
			if job == nil {
				continue
			}

			log.Printf("🐾 Processing job %s [Type: %s, Attempt: %d]...", job.ID, job.Type, job.Attempts)

			// Execute the worker payload processing
			err = s.processJob(ctx, job)
			if err != nil {
				log.Printf("❌ Job %s failed: %v", job.ID, err)
				_ = s.q.Nack(ctx, job.ID, err)
			} else {
				log.Printf("✅ Job %s completed successfully", job.ID)
				_ = s.q.Ack(ctx, job.ID)
			}
		}
	}
}

func (s *Server) processJob(ctx context.Context, job *queue.Job) error {
	// Emulate executing the Python triage/fixing worker
	time.Sleep(2 * time.Second)
	log.Printf("🤖 Worker: Spawning Python script to process payload of size %d bytes...", len(job.Payload))
	return nil
}
