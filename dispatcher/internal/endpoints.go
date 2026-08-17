package internal

import (
	"context"
	"shared"

	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

// encapsulates shared long term dependencies for http server
type ServerEnv struct {
	ctx  *context.Context
	s3m  *shared.S3Manager
	rmqm *shared.RMQManager

	bucket string
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(ErrorResponse{
		Error: err.Error(),
	})
}

func (env *ServerEnv) ResponseRootHanlder(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"message": "AlpineJudge Alive"})
}

func (env *ServerEnv) SubmissionReciever(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, fmt.Errorf("Method not allowed"))
		return
	}

	var submission SubmissionSpec

	// malformed submission
	err := json.NewDecoder(r.Body).Decode(&submission)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	/*
		Using the incoming request's context so if the user closes the browser early,
		downstream validations stop immediately.
	*/
	if err = ValidateSubmission(r.Context(), *env.s3m, submission); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	// marshall requests into transferrable SubmissionSpec
	bodyBytes, err := json.Marshal(submission)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	//package message data frame for RabbitMQ delivery
	msg := amqp.Publishing{
		ContentType: "application/json",
		MessageId:   submission.SubmissionID,
		Body:        bodyBytes,
	}

	if err := env.rmqm.Publish(
		r.Context(),
		os.Getenv("RABBITMQ_QUEUE_NAME"),
		msg,
	); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("Message broker drop: %v", err))
		return
	}

	// successful submission
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "Queued", "submission_id": submission.SubmissionID,
	})
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
	submissionID := r.PathValue("submission_id")
	routingKey := submissionID

	exchangeName, exists := os.LookupEnv("DIRECT_EXCHANGE_NAME")
	if !exists {
		log.Fatal("Env var DIRECT_EXCHANGE_NAME not found")
	}

	// 2. Subscribe and bind the temp queue to the exchange using the routing key
	execEventQueue := make(chan amqp.Delivery)
	if err := env.rmqm.SubscribeToExchange(
		*env.ctx,
		execEventQueue,
		exchangeName,
		routingKey, // <-- ONLY listen for messages matching submission_id
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

			// write to HTTP pipe
			if _, err := fmt.Fprintf(w, "data: %s\n\n", string(msg.Body)); err != nil {
				// if write fails, DO NOT ACK. Nack/Reject or exit loop to trigger channel close.
				_ = msg.Nack(false, false) // Rejects message without requeueing
				return
			}
			flusher.Flush()    // flush bytes on the network
			_ = msg.Ack(false) // manual ACK
		}
	}
}

func InitHTTPServer(
	ctx context.Context, s3m *shared.S3Manager, rmqm *shared.RMQManager,
) *http.Server {

	env := &ServerEnv{
		ctx:    &ctx,
		s3m:    s3m,
		rmqm:   rmqm,
		bucket: os.Getenv("MINIO_S3_BUCKET"),
	}

	mux := http.NewServeMux()

	// {$} make matches ONLY the exact root path "/"
	mux.HandleFunc("GET /{$}", env.ResponseRootHanlder)
	mux.HandleFunc("POST /submit", env.SubmissionReciever)
	mux.HandleFunc("GET /submissions/{submission_id}/events", env.SSEHandler)

	serverPort := ":1111"
	fmt.Printf("Starting server on http://localhost%s\n", serverPort)

	return &http.Server{
		Addr:    serverPort,
		Handler: mux,
	}
}
