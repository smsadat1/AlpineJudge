package internal

import (
	"fmt"
	"os"
	"path/filepath"
)

func findTestcaseFiles(testsetPath string, i int) (string, string, bool) {

	// Try standard integer first, then various padded formats
	formats := []string{
		fmt.Sprintf("%d", i),   // 1.in
		fmt.Sprintf("%02d", i), // 01.in
		fmt.Sprintf("%03d", i), // 001.in
		fmt.Sprintf("%04d", i), // 0001.in
		fmt.Sprintf("%05d", i), // 00001.in
		fmt.Sprintf("%06d", i), // 000001.in
	}

	for _, name := range formats {
		input := filepath.Join(testsetPath, name+".in")
		output := filepath.Join(testsetPath, name+".out")

		if _, err := os.Stat(input); err == nil {
			if _, errOut := os.Stat(output); errOut == nil {
				return input, output, true
			}
		}
	}

	return "", "", false
}
