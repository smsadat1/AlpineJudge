package tests

import (
	"fmt"
	"shared"
	"testing"
)

// Original "SubmissionSpec" is in dispatcher, copied here for shared infra test purpose
type SubmissionSpec_Copy struct {
	SubmissionID string `json:"submission_id"`
	Bucket       string `json:"bucket"`
	Language     string `json:"language"`
	Source       string `json:"source"`
	Testset      string `json:"testset"`
}

type TestRepository struct {
	TestSubmSpec SubmissionSpec_Copy
	TestJobSpec  shared.JobSpec
}

func NewRepository(t *testing.T) TestRepository {
	t.Helper()

	tsID := "test001"
	testsetID := "ts001"

	tss := SubmissionSpec_Copy{
		SubmissionID: "testsub001",
		Bucket:       "testbucket",
		Language:     "cpp",
		Source:       `#include<iostream> int main() {return 0;}`,
		Testset:      "ts001",
	}

	tjs := shared.JobSpec{
		Language: "cpp",
		Version:  "c++17",
		Image:    "ghcr.io/smsadat1/alpinejudge/master:test",
		Source: `
		#include <iostream>

		int main() {
			
		    int a, b = 0;
		    std::cin >> a >> b;
		    std::cout << a + b;
			
		    return 0;   
		}`,
		SubmissionID:         tsID,
		Bucket:               "testbucket",
		SrcCodeS3Key:         fmt.Sprintf("submissions/%s/main.cpp", tsID),
		TestsetS3Key:         fmt.Sprintf("testsets/%s/main.cpp", tsID),
		Testset:              testsetID,
		TestsetVersion:       "v1",
		SSEQueue:             "queue-001",
		PerTestMemoryLimitMB: 1024,
		PerTestLogLimitKB:    512,
		PerTestTimeoutsec:    5,
	}

	return TestRepository{
		TestSubmSpec: tss,
		TestJobSpec:  tjs,
	}
}
