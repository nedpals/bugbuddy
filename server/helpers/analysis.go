package helpers

import (
	"io/fs"

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
	Exp      string
	Output   string
	Err      error
}

func (res AnalyzerResult) Stats() (int, int, error) {
	return GetAnalyzerStats(res.Template, res.Exp, res.Err)
}

func GetAnalyzerStats(template *eg.CompiledErrorTemplate, exp string, err error) (int, int, error) {
	recognized := 0
	processed := 0

	if template != nil && template != eg.FallbackErrorTemplate {
		recognized++
	}

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
		if err.Error() == "template not found" {
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
