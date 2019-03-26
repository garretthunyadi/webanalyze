package webanalyze

import (
	"fmt"
	"testing"
)

func TestProcess(t *testing.T) {
	// funct process(job *Job) ([]Match, error)
	fmt.Println("here")

	analyzer, err := NewWebAnalyzer(4, "apps.json")
	if err != nil {
		t.Error("Unexpected 1")
	}

	analyzer.Schedule(NewOnlineJob("google.com", "", nil))

	for result := range analyzer.Results {
		fmt.Printf("%#v\n", result)
	}
}

// func TestFailure(t *testing.T) {
// 	total := 60
// 	if total != 10 {
// 		t.Errorf("Sum was incorrect, got: %d, want: %d.", total, 10)
// 	}
// }
