package runner

type RunCommand struct {
	Universal []string // Command if both OS have same command. Not required if either Windows or Unix OS is specified.
	Unix      []string // Command for Unix-based systems (eg. macOS, Linux). Not required if Universal is specified.
	Windows   []string // Command for Windows-based systems. Not required if Universal is specified.
}

var defaultRunCommands = map[string]RunCommand{
	"python":       {Universal: []string{"python3 ${filename}"}},
	"c":            {Universal: []string{"gcc -o ${fileNoExt} ${filename}", "./${fileNoExt}"}},
	"cpp":          {Universal: []string{"g++ -o ${fileNoExt} ${filename}", "./${fileNoExt}"}},
	"java":         {Universal: []string{"javac ${filename}", "java ${filenameNoExt}"}},
	"rust":         {Universal: []string{"rustc ${filename}", "./${fileNoExt}"}},
	"go":           {Universal: []string{"go run ${filename}"}},
	"js":           {Universal: []string{"node ${filename}"}},
	"typescript":   {Universal: []string{"ts-node ${filename}"}},
	"php":          {Universal: []string{"php ${filename}"}},
	"ruby":         {Universal: []string{"ruby ${filename}"}},
	"perl":         {Universal: []string{"perl ${filename}"}},
	"bash":         {Universal: []string{"bash ${filename}"}},
	"sh":           {Universal: []string{"sh ${filename}"}},
	"zsh":          {Universal: []string{"zsh ${filename}"}},
	"powershell":   {Universal: []string{"powershell -ExecutionPolicy Bypass -File ${filename}"}},
	"batch":        {Universal: []string{"cmd /c ${filename}"}},
	"lua":          {Universal: []string{"lua ${filename}"}},
	"r":            {Universal: []string{"Rscript ${filename}"}},
	"dart":         {Universal: []string{"dart ${filename}"}},
	"elixir":       {Universal: []string{"elixir ${filename}"}},
	"erlang":       {Universal: []string{"erl -noshell -s ${fileNoExt} main -s init stop"}},
	"clojure":      {Universal: []string{"clojure ${filename}"}},
	"julia":        {Universal: []string{"julia ${filename}"}},
	"coffeescript": {Universal: []string{"coffee ${filename}"}},
	"crystal":      {Universal: []string{"crystal ${filename}"}},
	"nim":          {Universal: []string{"nim c -r ${filename}"}},
	"ocaml":        {Universal: []string{"ocaml ${filename}"}},
	"pascal":       {Universal: []string{"fpc ${filename}", "./${fileNoExt}"}},
	"perl6":        {Universal: []string{"perl6 ${filename}"}},
	"prolog":       {Universal: []string{"swipl -q -t main -f ${filename}"}},
	"racket":       {Universal: []string{"racket ${filename}"}},
	"raku":         {Universal: []string{"raku ${filename}"}},
	"reason":       {Universal: []string{"refmt ${filename}", "node ${fileNoExt}.js"}},
	"red":          {Universal: []string{"red ${filename}"}},
	"solidity":     {Universal: []string{"solc ${filename}"}},
	"swift":        {Universal: []string{"swift ${filename}"}},
	"v":            {Universal: []string{"v run ${filename}"}},
	"vb":           {Universal: []string{"vbnc ${filename}", "mono ${fileNoExt}.exe"}},
	"vbnet":        {Universal: []string{"vbnc ${filename}", "mono ${fileNoExt}.exe"}},
	"vbs":          {Universal: []string{"cscript ${filename}"}},
	"zig":          {Universal: []string{"zig run ${filename}"}},
}
