package main

import (
	"bytes"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

var eventHandlerTmpl = template.Must(template.New("eventHandler").Funcs(template.FuncMap{
	"constName":      constName,
	"isDiscordEvent": isDiscordEvent,
	"privateName":    privateName,
}).Parse(`// Code generated by \"eventhandlers\"; DO NOT EDIT
// See events.go
package discord
// Following are all the event types.
// Event type values are used to match the events returned by Discord.
// EventTypes surrounded by __ are synthetic and are internal to DiscordGo.
const ({{range .}}
  {{privateName .}}EventType = "{{constName .}}"{{end}}
)
{{range .}}
// {{privateName .}}EventHandler is an event handler for {{.}} events.
type {{privateName .}}EventHandler func(*Session, *{{.}})
// Type returns the event type for {{.}} events.
func (eh {{privateName .}}EventHandler) Type() string {
  return {{privateName .}}EventType
}
{{if isDiscordEvent .}}
// New returns a new instance of {{.}}.
func (eh {{privateName .}}EventHandler) New() interface{} {
  return &{{.}}{}
}{{end}}
// Handle is the handler for {{.}} events.
func (eh {{privateName .}}EventHandler) Handle(s *Session, i interface{}) {
  if t, ok := i.(*{{.}}); ok {
    eh(s, t)
  }
}
{{end}}
func handlerForInterface(handler interface{}) EventHandler {
  switch v := handler.(type) {
  case func(*Session, interface{}):
    return interfaceEventHandler(v){{range .}}
  case func(*Session, *{{.}}):
    return {{privateName .}}EventHandler(v){{end}}
  }
  return nil
}
func init() { {{range .}}{{if isDiscordEvent .}}
  registerInterfaceProvider({{privateName .}}EventHandler(nil)){{end}}{{end}}
}
`))

func main() {
	var buf bytes.Buffer
	dir := filepath.Dir("./api/discord/")

	fs := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fs, "./api/discord/events.go", nil, 0)
	if err != nil {
		log.Fatalf("warning: internal error: could not parse events.go: %s", err)
		return
	}

	names := []string{}
	for object := range parsedFile.Scope.Objects {
		names = append(names, object)
	}
	sort.Strings(names)
	eventHandlerTmpl.Execute(&buf, names)

	src, err := format.Source(buf.Bytes())
	if err != nil {
		log.Println("warning: internal error: invalid Go generated:", err)
		src = buf.Bytes()
	}

	err = ioutil.WriteFile(filepath.Join(dir, strings.ToLower("eventhandlers.go")), src, 0644)
	if err != nil {
		log.Fatal(buf, "writing output: %s", err)
	}
}

var constRegexp = regexp.MustCompile("([a-z])([A-Z])")

func constCase(name string) string {
	return strings.ToUpper(constRegexp.ReplaceAllString(name, "${1}_${2}"))
}

func isDiscordEvent(name string) bool {
	switch {
	case name == "Connect", name == "Disconnect", name == "Event", name == "RateLimit", name == "Interface":
		return false
	default:
		return true
	}
}

func constName(name string) string {
	if !isDiscordEvent(name) {
		return "__" + constCase(name) + "__"
	}

	return constCase(name)
}

func privateName(name string) string {
	return strings.ToLower(string(name[0])) + name[1:]
}
