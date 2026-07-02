package dispatcher

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

var (
	ResponseRootHanlderFn = ResponseRootHanlder
	SubmissionRecieverFn  = SubmissionReciever
	ValidateSubmissionFn  = validateSubmission
)

func ResponseRootHanlder(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "AlpineJudge active")
}

func SubmissionReciever(w http.ResponseWriter, r *http.Request) {
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
	if err = ValidateSubmissionFn(submission); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// successful submission
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func InitHttpServer() {

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", ResponseRootHanlderFn)
	mux.HandleFunc("POST /submit", SubmissionRecieverFn)
	// mux.HandleFunc("GET /job/{job_id}/events")
	// mux.HandleFunc("GET /jobs/{job_id}/result")

	serverPort := "8080"
	fmt.Printf("Starting server on http://localhost%s\n", serverPort)

	if err := http.ListenAndServe(serverPort, mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
