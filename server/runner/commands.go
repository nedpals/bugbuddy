package runner

type RunCommand struct {
	Universal string // Command if both OS have same command. Not required if either Windows or Unix OS is specified.
	Unix      string // Command for Unix-based systems (eg. macOS, Linux). Not required if Universal is specified.
	Windows   string // Command for Windows-based systems. Not required if Universal is specified.
}

var defaultRunCommands = map[string]RunCommand{
	"python":       {Universal: "python3 ${filename}"},
	"c":            {Universal: "gcc ${filename} -o ${fileNoExt} && ./${fileNoExt}"},
	"cpp":          {Universal: "g++ ${filename} -o ${fileNoExt} && ./${fileNoExt}"},
	"java":         {Universal: "javac ${filename} && java ${filenameNoExt}"},
	"rust":         {Universal: "rustc ${filename} && ./${fileNoExt}"},
	"go":           {Universal: "go run ${filename}"},
	"js":           {Universal: "node ${filename}"},
	"typescript":   {Universal: "ts-node ${filename}"},
	"php":          {Universal: "php ${filename}"},
	"ruby":         {Universal: "ruby ${filename}"},
	"perl":         {Universal: "perl ${filename}"},
	"bash":         {Universal: "bash ${filename}"},
	"sh":           {Universal: "sh ${filename}"},
	"zsh":          {Universal: "zsh ${filename}"},
	"powershell":   {Universal: "powershell -ExecutionPolicy Bypass -File ${filename}"},
	"batch":        {Universal: "cmd /c ${filename}"},
	"lua":          {Universal: "lua ${filename}"},
	"r":            {Universal: "Rscript ${filename}"},
	"dart":         {Universal: "dart ${filename}"},
	"elixir":       {Universal: "elixir ${filename}"},
	"erlang":       {Universal: "erl -noshell -s ${fileNoExt} main -s init stop"},
	"clojure":      {Universal: "clojure ${filename}"},
	"julia":        {Universal: "julia ${filename}"},
	"coffeescript": {Universal: "coffee ${filename}"},
	"crystal":      {Universal: "crystal ${filename}"},
	"nim":          {Universal: "nim c -r ${filename}"},
	"ocaml":        {Universal: "ocaml ${filename}"},
	"pascal":       {Universal: "fpc ${filename} && ./${fileNoExt}"},
	"perl6":        {Universal: "perl6 ${filename}"},
	"prolog":       {Universal: "swipl -q -t main -f ${filename}"},
	"racket":       {Universal: "racket ${filename}"},
	"raku":         {Universal: "raku ${filename}"},
	"reason":       {Universal: "refmt ${filename} && node ${fileNoExt}.js"},
	"red":          {Universal: "red ${filename}"},
	"solidity":     {Universal: "solc ${filename}"},
	"swift":        {Universal: "swift ${filename}"},
	"v":            {Universal: "v run ${filename}"},
	"vb":           {Universal: "vbnc ${filename} && mono ${fileNoExt}.exe"},
	"vbnet":        {Universal: "vbnc ${filename} && mono ${fileNoExt}.exe"},
	"vbs":          {Universal: "cscript ${filename}"},
	"zig":          {Universal: "zig run ${filename}"},
}
