package tests

import (
	"bytes"
	"context"
	"dispatcher"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"shared"
	"testing"
	"time"
)

func Test_HTTPServer_Integration(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()

	t.Setenv("TEST_RMQ_URL", "amqp://guest:guest@localhost:5672/")
	t.Setenv("TEST_S3_URL", "http://localhost:9000")
	t.Setenv("TEST_S3_USERNAME", "minioadmin")
	t.Setenv("TEST_S3_PASSWORD", "minioadminpassword")
	t.Setenv("TEST_S3_BUCKET_NAME", "ajbucket-test-http")
	t.Setenv("TEST_S3_REGION_NAME", "us-east-1")

	s3m, err := shared.InitS3Manager(
		ctx,
		os.Getenv("TEST_S3_BUCKET_NAME"),
		os.Getenv("TEST_S3_REGION_NAME"),
		os.Getenv("TEST_S3_USERNAME"),
		os.Getenv("TEST_S3_PASSWORD"),
		os.Getenv("TEST_S3_URL"),
	)

	if err != nil {
		t.Error(err)
	}

	if err := dispatcher.LoadConfigs("artifacts/config.example.yaml"); err != nil {
		t.Fatal(err)
	}

	_, _ = s3m.CreateABucket(ctx, os.Getenv("TEST_S3_BUCKET_NAME"))

	// upload test artifact
	fileData, err := os.Open("artifacts/result.json")
	if err != nil {
		t.Error(err)
	}

	s3m.UploadFileToS3(ctx, "/submission/s010/result.json", fileData)
	s3m.UploadDirToS3(ctx, "ts001/v1", "artifacts/ts001")

	rmqMgr, err := shared.NewRMQManager(ctx, os.Getenv("TEST_RMQ_URL"))
	if err != nil {
		t.Fatalf("Failed to link RMQ infrastructure: %v", err)
	}
	t.Cleanup(func() { rmqMgr.Close() })

	serverConfig := dispatcher.InitHTTPServer(ctx, s3m, rmqMgr)

	// httptest.NewServer boots up the mux handler on a random, open local port automatically
	testServer := httptest.NewServer(serverConfig.Handler)
	t.Cleanup(testServer.Close)

	// test 1
	t.Run("GET / Gives alive response", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/")
		if err != nil {
			t.Error(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 Accepted, got: %d\n", resp.StatusCode)
		}

		var jsonResp map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&jsonResp)
		if jsonResp["message"] != "AlpineJudge Alive" {
			t.Errorf("Expected status 'queued', got payload: %v", jsonResp)
		}
	})

	// test 2
	t.Run("POST /submit Sends payload to RabbitMQ", func(t *testing.T) {
		submission := dispatcher.SubmissionSpec{
			SubmissionID:   "s011",
			Bucket:         os.Getenv("TEST_S3_BUCKET_NAME"),
			Language:       "cpp",
			Version:        "c++17",
			Source:         `#include<stdion.h>\nint main() \n{ std::cout << "Hello World\n";\n}`,
			Testset:        "ts001",
			TestsetVersion: "v1",
		}

		bodyBytes, _ := json.Marshal(submission)

		// send actual HTTP request directly to the live test server URL
		resp, err := http.Post(testServer.URL+"/submit", "application/json", bytes.NewBuffer(bodyBytes))
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusAccepted {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 202 Accepted, got: %d\nMessage: %v\n", resp.StatusCode, string(body))
		}

		var jsonResp map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&jsonResp)
		if jsonResp["status"] != "Queued" {
			t.Errorf("Expected status 'queued', got payload: %v", jsonResp)
		}
	})

	// test3
	t.Run("GET /jobs/{submission_id}/result Extracts route key variables", func(t *testing.T) {
		targetSubmissionID := "s0101"

		// Run a GET against the pattern matched route
		resp, err := http.Get(testServer.URL + "/submissions/" + targetSubmissionID + "/result")
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
		defer resp.Body.Close()

		/*
			Expects DownloadFileFromS3 to fail (status 500) if the file doesn't exist,
			But it proves path router extracted the variable properly
		*/
		if resp.StatusCode == http.StatusBadRequest {
			t.Error("Routing failed: engine did not parse submission_id path parameter variable")
		}
	})

}
