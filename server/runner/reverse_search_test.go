package runner_test

import (
	"testing"

	"github.com/nedpals/bugbuddy/server/runner"
)

func TestGetIdAndPathFromCommand(t *testing.T) {
	testCases := []struct {
		Command      string
		ExpectedID   string
		ExpectedPath string
	}{
		{
			Command:      "python3 main.py",
			ExpectedID:   "python",
			ExpectedPath: "main.py",
		},
		{
			Command:      "go run main.go",
			ExpectedID:   "go",
			ExpectedPath: "main.go",
		},
		{
			Command:      "python3 -m http.server 8080 main.py",
			ExpectedID:   "python",
			ExpectedPath: "main.py",
		},
	}

	for _, tc := range testCases {
		languageID, path := runner.GetIdAndPathFromCommand(tc.Command)

		if languageID != tc.ExpectedID {
			t.Errorf("Expected language ID %s, but got %s", tc.ExpectedID, languageID)
		}

		if path != tc.ExpectedPath {
			t.Errorf("Expected path %s, but got %s", tc.ExpectedPath, path)
		}
	}
}
