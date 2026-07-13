package tests

import (
	"dispatcher"
	"slices"
	"testing"
)

func Test_CfgManager(t *testing.T) {

	// reset global map state to prevent cross-test contamination
	dispatcher.AvailableLanguages = make(map[string]map[string]struct{})

	if err := dispatcher.LoadConfigs("artifacts/config.example.yaml"); err != nil {
		t.Logf("%v", err)
	}

	cVersions := dispatcher.AvailableLanguages["c"]
	cppVersions := dispatcher.AvailableLanguages["cpp"]
	goVersions := dispatcher.AvailableLanguages["go"]
	javaVersions := dispatcher.AvailableLanguages["java"]
	nodeVersions := dispatcher.AvailableLanguages["node"]
	pythonVersions := dispatcher.AvailableLanguages["python"]

	_, hasC99 := cVersions["c99"]
	_, hasC11 := cVersions["c11"]
	_, hasC17 := cVersions["c17"]

	if len(cVersions) != 3 || !hasC99 || !hasC11 || !hasC17 {
		t.Error("c versions mismatched")
	}

	_, hasCpp11 := cppVersions["c++11"]
	_, hasCpp17 := cppVersions["c++17"]
	_, hasCpp20 := cppVersions["c++20"]

	if len(cppVersions) != 3 || !hasCpp11 || !hasCpp17 || !hasCpp20 {
		t.Error("cpp versions mismatched")
	}

	_, hasGo124 := goVersions["go1.24"]
	_, hasGo126 := goVersions["go1.26"]

	if len(goVersions) != 2 || !hasGo124 || !hasGo126 {
		t.Error("go versions mismatched")
	}

	_, hasJava14 := javaVersions["java25"]
	_, hasJava16 := javaVersions["java26"]

	if len(javaVersions) != 2 || !hasJava14 || !hasJava16 {
		t.Error("java versions mismatched")
	}

	_, hasNode18 := nodeVersions["node18"]
	_, hasNode22 := nodeVersions["node22"]

	if len(nodeVersions) != 2 || !hasNode18 || !hasNode22 {
		t.Error("node versions mismatched")
	}

	_, hasPython310 := pythonVersions["python3.10"]
	_, hasPython312 := pythonVersions["python3.12"]

	if len(pythonVersions) != 2 || !hasPython310 || !hasPython312 {
		t.Error("python versions mismatched")
	}

	availRunners := dispatcher.AvailableRunners
	expectedRunners := []string{"runner-001", "runner-002"}

	slices.Sort(availRunners)
	slices.Sort(expectedRunners)

	if len(availRunners) != 2 || !slices.Equal(availRunners, expectedRunners) {
		t.Error("runner availability mismatched")
	}
}
