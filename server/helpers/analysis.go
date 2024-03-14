package helpers

import (
	"io/fs"
	"strings"

	eg "github.com/nedpals/errgoengine"
	"github.com/nedpals/errgoengine/error_templates"
)

func DefaultEngine() *eg.ErrgoEngine {
	engine := &eg.ErrgoEngine{
		ErrorTemplates: eg.ErrorTemplates{},
		FS: &eg.MultiReadFileFS{
			FSs: []fs.ReadFileFS{
				NewSharedFS(),
			},
		},
		SharedStore: eg.NewEmptyStore(),
		OutputGen:   &eg.OutputGenerator{},
	}

	error_templates.LoadErrorTemplates(&engine.ErrorTemplates)
	return engine
}

type AnalyzerResult struct {
	Template *eg.CompiledErrorTemplate
	Data     *eg.ContextData
	Exp      string // main explanation
	Output   string // full explanation / output
	Err      error
}

func (res AnalyzerResult) Stats() (int, int, error) {
	return GetAnalyzerStats(res.Template, res.Exp, res.Err)
}

func GetAnalyzerStats(template *eg.CompiledErrorTemplate, exp string, err error) (int, int, error) {
	recognized := 0
	processed := 0

	// mark as recognized if the template is not nil and not the fallback error template
	if template != nil && template != eg.FallbackErrorTemplate {
		recognized++
	}

	// mark as processed if the error is nil and the exp is not empty
	if err == nil && len(exp) > 0 {
		processed++
	}

	return recognized, processed, err
}

func AnalyzeError(engine *eg.ErrgoEngine, workingDir string, msg string) (res AnalyzerResult) {
	defer func() {
		if r := recover(); r != nil {
			res.Err = r.(error)
			return
		}
	}()

	t, d, err := engine.Analyze(workingDir, msg)
	res.Template = t
	res.Data = d
	res.Err = err

	if err != nil {
		// return fallback error template if there is no template found
		if t == nil && strings.HasPrefix(err.Error(), "template not found") {
			t = eg.FallbackErrorTemplate
		} else {
			return
		}
	}

	e, o := engine.Translate(t, d)

	res.Exp = e
	res.Output = o
	res.Err = err

	return res
}
