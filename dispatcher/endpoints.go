package dispatcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	amqp "github.com/rabbitmq/amqp091-go"

	"shared"
)

// encapsulates shared long term dependencies for http server
type ServerEnv struct {
	ctx  *context.Context
	s3m  *shared.S3Manager
	rmqm *shared.RMQManager
}

func (env *ServerEnv) ResponseRootHanlder(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "AlpineJudge active")
}

func (env *ServerEnv) SubmissionReciever(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var submission SubmissionSpec

	// malformed submission
	err := json.NewDecoder(r.Body).Decode(&submission)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// wrong submission
	if err = ValidateSubmission(*env.ctx, *env.s3m, submission); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// successful submission
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (env *ServerEnv) SSEHandler(w http.ResponseWriter, r *http.Request) {

	// important SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*") // change in PROD

	// flush write
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	execEventQueue := make(chan amqp.Delivery)
	if err := env.rmqm.Subscribe(*env.ctx, execEventQueue); err != nil {
		http.Error(w, "Execution event queue failed!", http.StatusInternalServerError)
		return
	}

	for {
		select {
		case <-r.Context().Done():
			fmt.Println("Client disconnected.")
			return

		case e := <-execEventQueue:
			// Format required by SSE: "data: <message>\n\n"
			// double newline indicating the end of an event block
			fmt.Fprintf(w, "Event: %v\n\n", e)

			// flush immediately
			flusher.Flush()
		}
	}
}

func (env *ServerEnv) ResultReciever(w http.ResponseWriter, r *http.Request) {

	bucket := "ajbucket" // replace with env derived
	job_id := "example"  // replace with url derived
	key := "/submission/" + job_id + "/result.json"
	resultFile := "result.json"

	if err := env.s3m.DownloadFileFromS3(*env.ctx, bucket, key, resultFile); err != nil {
		http.Error(w, "Failed fetching result\n", http.StatusInternalServerError)
		return
	}
}

func InitHttpServer(ctx context.Context, s3m *shared.S3Manager, rmqm *shared.RMQManager) {

	env := &ServerEnv{
		ctx:  &ctx,
		s3m:  s3m,
		rmqm: rmqm,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", env.ResponseRootHanlder)
	mux.HandleFunc("POST /submit", env.SubmissionReciever)
	mux.HandleFunc("GET /job/{job_id}/events", env.SSEHandler)
	mux.HandleFunc("GET /jobs/{job_id}/result", env.ResultReciever)

	serverPort := ":8080"
	fmt.Printf("Starting server on http://localhost%s\n", serverPort)

	if err := http.ListenAndServe(serverPort, mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
