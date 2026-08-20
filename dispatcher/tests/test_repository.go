package tests

import (
	"dispatcher/internal"
	"shared"
	"testing"
)

type TestRepository struct {
	TestSubmSpec internal.SubmissionSpec
	TestJobSpec  shared.JobSpec
}

func NewDispatcherRepository(t *testing.T) TestRepository {
	t.Helper()

	tsID := "test001"
	testsetID := "ts001"

	tss := internal.SubmissionSpec{
		SubmissionID: "testsub001",
		Language:     "cpp",
		Source:       `#include<iostream> int main() {return 0;}`,
		Testset:      "ts001",
	}

	tjs := shared.JobSpec{
		SubmissionID: tsID,
		Language:     "cpp",
		Source: `
		#include <iostream>

		int main() {
			
		    int a, b = 0;
		    std::cin >> a >> b;
		    std::cout << a + b;
			
		    return 0;   
		}`,
		Bucket:               "testbucket",
		Testset:              testsetID,
		PerTestMemoryLimitMB: 1024,
		PerTestLogLimitKB:    512,
		PerTestTimeoutsec:    5,
	}

	return TestRepository{
		TestSubmSpec: tss,
		TestJobSpec:  tjs,
	}
}
