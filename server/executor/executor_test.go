package executor_test

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/nedpals/bugbuddy/server/executor"
	"github.com/nedpals/bugbuddy/server/helpers"
	"github.com/nedpals/errgoengine"
)

type TestCollector struct {
	Engine   *errgoengine.ErrgoEngine
	ExitCode int
	Outputs  []string
}

func (tc *TestCollector) Collect(exitCode int, args, workingDir, stderr string) (int, int, error) {
	tc.ExitCode = exitCode
	result := helpers.AnalyzeError(tc.Engine, workingDir, stderr)
	r, p, err := result.Stats()
	if r > 0 {
		tc.Outputs = append(tc.Outputs, stderr)
	}
	return r, p, err
}

var cases = []struct {
	BeforeInput [][]string
	Inputs      [][]string
	AfterInput  [][]string
	ErrorCounts []int
	Outputs     []string
}{
	{
		Inputs: [][]string{
			{"python3", "./test_programs/simple.py"},
		},
		ErrorCounts: []int{1},
		Outputs: []string{
			`
Traceback (most recent call last):
  File "./test_programs/simple.py", line 4, in <module>
    print(name)
          ^^^^
NameError: name 'name' is not defined
`,
		},
	},
	{
		Inputs: [][]string{
			{"python3", "./test_programs/complex.py"},
		},
		ErrorCounts: []int{2},
		Outputs: []string{
			`
Traceback (most recent call last):
  File "./test_programs/complex.py", line 1, in <module>
    print(name)
          ^^^^
NameError: name 'name' is not defined
`,
			`
Traceback (most recent call last):
  File "./test_programs/complex.py", line 6, in <module>
    print(a / 0)
          ~~^~~
ZeroDivisionError: division by zero`,
		},
	},
	{
		Inputs: [][]string{
			{"python3", "./test_programs/dangling.py"},
		},
		ErrorCounts: []int{1},
		Outputs: []string{
			`
Ooops I have been included in error
Traceback (most recent call last):
  File "./test_programs/dangling.py", line 1, in <module>
    print(name)
          ^^^^
NameError: name 'name' is not defined
`,
		},
	},
}

func TestExecute(t *testing.T) {
	engine := helpers.DefaultEngine()
	collector := &TestCollector{Engine: engine}

	for _, c := range cases {
		for _, inputs := range c.BeforeInput {
			cmd := exec.Command(inputs[0], inputs[1:]...)
			if err := cmd.Run(); err != nil {
				t.Fatal(err)
			}
		}

		for i, inputs := range c.Inputs {
			t.Run(strings.Join(inputs, " "), func(t *testing.T) {
				numErrors, _, err := executor.Execute(".", collector, inputs[0], inputs[1:]...)
				if err != nil {
					t.Fatal(err)
				}

				if numErrors != c.ErrorCounts[i] {
					t.Fatalf("expected %d errors, got %d", c.ErrorCounts[i], numErrors)
				}

				if len(collector.Outputs) != len(c.Outputs) {
					t.Fatalf("expected %d outputs, got %d", len(c.Outputs), len(collector.Outputs))
				}

				for i, output := range collector.Outputs {
					trimmedOut := strings.TrimSpace(output)
					trimmedExp := strings.TrimSpace(c.Outputs[i])

					if trimmedOut != trimmedExp {
						fmt.Println([]byte(trimmedOut), []byte(trimmedExp))

						t.Fatalf("expected %s, got %s", trimmedExp, trimmedOut)
					}
				}
			})

			collector.Outputs = []string{}
		}

		for _, inputs := range c.AfterInput {
			cmd := exec.Command(inputs[0], inputs[1:]...)
			if err := cmd.Run(); err != nil {
				t.Fatal(err)
			}
		}
	}
}