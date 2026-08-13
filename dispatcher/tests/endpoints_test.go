package tests

import (
	"bytes"
	"context"
	"dispatcher/internal"
	"encoding/json"
	"io"
	"local/testrunner/factory"
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

	tr := NewDispatcherRepository(t)
	submissionSpec := tr.TestSubmSpec

	t.Setenv("TEST_S3_BUCKET_NAME", "dispatcherTestBucket")
	_, _ = tf.S3m.CreateABucket(ctx, os.Getenv("TEST_S3_BUCKET_NAME"))

	tf.S3m.UploadDirToS3(ctx, "ts001/v1", "ts001")

	serverConfig := internal.InitHTTPServer(ctx, tf.S3m, tf.Rmqm)

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
}
