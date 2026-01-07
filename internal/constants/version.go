package constants

import (
	"runtime/debug"
	"strings"
	"text/template"
)

var (
	CLIVersion        = "no-info"
	VerboseCLIVersion = ""
)

type versionStruct struct {
	Version   string
	GoVersion string
	Time      string
	Commit    string
	OS        string
	Arch      string
	Modified  bool
}

const (
	verboseTemplate = `Version: {{.Version}}
Go Version: {{.GoVersion}}
Git Commit: {{.Commit}}
Commit Date: {{.Time}}
OS/Arch: {{.OS}}/{{.Arch}}
Dirty: {{.Modified}}`
)

func init() {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	var vs versionStruct

	vs.Version = CLIVersion
	vs.GoVersion = bi.GoVersion

	for _, kv := range bi.Settings {
		switch kv.Key {
		case "vcs.time":
			vs.Time = kv.Value
		case "vcs.revision":
			vs.Commit = kv.Value
		case "vcs.modified":
			vs.Modified = kv.Value == "true"
		case "GOOS":
			vs.OS = kv.Value
		case "GOARCH":
			vs.Arch = kv.Value
		}
	}

	VerboseCLIVersion = vs.String()
}

func (vs *versionStruct) String() string {
	stringBuilder := &strings.Builder{}
	tmpl := template.Must(template.New("version").Parse(verboseTemplate))

	err := tmpl.Execute(stringBuilder, vs)
	if err != nil {
		panic(err)
	}

	return stringBuilder.String()
}
