package dispatcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"shared"
)

// encapsulates shared long term dependencies for http server
type ServerEnv struct {
	ctx  *context.Context
	s3m  *shared.S3Manager
	rmqm *shared.RMQManager

	bucket string
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

	/*
		Using the incoming request's context so if the user closes the browser early,
		downstream validations stop immediately.
	*/
	if err = ValidateSubmission(r.Context(), *env.s3m, submission); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// marshall requests into transferrable SubmissionSpec
	bodyBytes, err := json.Marshal(submission)
	if err != nil {
		http.Error(w, "Failed to encode submission", http.StatusInternalServerError)
		return
	}

	//package message data frame for RabbitMQ delivery
	msg := amqp.Publishing{
		ContentType: "application/json",
		MessageId:   fmt.Sprintf("sub_%d", time.Now().UnixNano()),
		Body:        bodyBytes,
	}

	if err := env.rmqm.Publish(
		r.Context(),
		os.Getenv("RABBITMQ_QUEUE_NAME"),
		msg,
	); err != nil {
		http.Error(w, fmt.Sprintf("Message broker drop: %v", err), http.StatusInternalServerError)
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

	// uniquely generated consumer tag to avoid multi-client registration conflicts
	uniqueConsumerTag := fmt.Sprintf("sse_%d", time.Now().UnixNano())
	execEventQueue := make(chan amqp.Delivery)
	if err := env.rmqm.Subscribe(
		*env.ctx,
		execEventQueue,
		uniqueConsumerTag,
		os.Getenv("RABBITMQ_QUEUE_NAME"),
	); err != nil {
		http.Error(w, "Execution event queue failed!", http.StatusInternalServerError)
		return
	}

	for {
		select {
		case <-r.Context().Done():
			log.Println("SSE client disconnected safely.")
			return
		case msg, ok := <-execEventQueue:
			if !ok {
				log.Println("Event queue subscription closed stream.")
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", string(msg.Body))
			flusher.Flush()

			_ = msg.Ack(false) // ACK message delivery processing
		}
	}
}

func (env *ServerEnv) ResultReciever(w http.ResponseWriter, r *http.Request) {

	jobID := r.PathValue("job_id")
	if jobID == "" {
		http.Error(w, "Missing job reference id parameter", http.StatusBadRequest)
		return
	}

	bucket := env.bucket
	key := "/submission/" + jobID + "/result.json"
	resultFile := fmt.Sprintf("/tmp/result_%s.json", jobID)

	/*
		TODO:
		Instead of writing to a local file then sending back.
		Directly stream back from S3 to client
	*/

	if err := env.s3m.DownloadFileFromS3(*env.ctx, bucket, key, resultFile); err != nil {
		http.Error(w, "Failed fetching result\n", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"fetched","job_id":"%s"}`, jobID)
}

func InitHTTPServer(
	ctx context.Context, s3m *shared.S3Manager, rmqm *shared.RMQManager,
) *http.Server {

	env := &ServerEnv{
		ctx:    &ctx,
		s3m:    s3m,
		rmqm:   rmqm,
		bucket: os.Getenv("S3_BUCKET_NAME"),
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", env.ResponseRootHanlder)
	mux.HandleFunc("POST /submit", env.SubmissionReciever)
	mux.HandleFunc("GET /job/{job_id}/events", env.SSEHandler)
	mux.HandleFunc("GET /jobs/{job_id}/result", env.ResultReciever)

	serverPort := ":8080"
	fmt.Printf("Starting server on http://localhost%s\n", serverPort)

	return &http.Server{
		Addr:    serverPort,
		Handler: mux,
	}
}
