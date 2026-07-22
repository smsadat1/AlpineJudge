package integration

import (
	"bytes"
	"context"
	"dispatcher"
	"encoding/json"
	"io"
	"local/testrunner/factory"
	"local/testrunner/repository"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func Test_HTTPServer_Integration(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tf := factory.NewTestFactory(t)
	tf.StartTestRMQ(t, ctx)
	tf.StartTestMinioS3(t, ctx)

	tr := repository.NewTestRepository(t)
	submissionSpec := tr.TestSubmSpec

	if err := dispatcher.LoadConfigs("../artifacts/config.example.yaml"); err != nil {
		t.Fatal(err)
	}

	_, _ = tf.S3m.CreateABucket(ctx, os.Getenv("TEST_S3_BUCKET_NAME"))

	// upload test artifact
	fileData, err := os.Open("../artifacts/result.json")
	if err != nil {
		t.Error(err)
	}

	tf.S3m.UploadFileToS3(ctx, "/submission/s010/result.json", fileData)
	tf.S3m.UploadDirToS3(ctx, "ts001/v1", "../artifacts/ts001")

	serverConfig := dispatcher.InitHTTPServer(ctx, tf.S3m, tf.Rmqm)

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

		bodyBytes, _ := json.Marshal(submissionSpec)

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
