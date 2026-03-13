package main

import (
	"fmt"
	"io/ioutil"
	"strings"
)

func main() {
	content, _ := ioutil.ReadFile("/Users/sunquan/.rick/jobs/perf_test_25/plan/task1.md")
	
	fmt.Printf("File content length: %d bytes\n", len(content))
	fmt.Printf("First 100 bytes: %q\n", string(content[:100]))
	
	// Check for section
	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "任务名称") {
			fmt.Printf("Line %d: %q (trimmed: %q)\n", i, line, trimmed)
		}
	}
}
