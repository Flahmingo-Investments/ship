package main

import (
	"text/template"

	"github.com/Flahmingo-Investments/ship/schema"
	pgs "github.com/lyft/protoc-gen-star"
	pgsgo "github.com/lyft/protoc-gen-star/lang/go"
)

// Compile time check to verify that EventifyPlugin satisfies the pgs.Module
// interface.
var _ pgs.Module = (*EventifyPlugin)(nil)

// EventifyPlugin adds ship Event interface to the messages.
type EventifyPlugin struct {
	*pgs.ModuleBase
	ctx pgsgo.Context
	tpl *template.Template
}

// Eventify returns an initialized EventifyPlugin.
func Eventify() *EventifyPlugin {
	// NOTE: we cannot initialize template here.
	// We need the the build context first to gather the required information.
	// E.g. package name, etc.
	//
	// Template will be initialized in InitContext.
	return &EventifyPlugin{
		ModuleBase: &pgs.ModuleBase{},
	}
}

// InitContext context initialises the build context.
// It will be initialized by the pgs.Generator.
func (p *EventifyPlugin) InitContext(buildCtx pgs.BuildContext) {
	p.ModuleBase.InitContext(buildCtx)
	p.ctx = pgsgo.InitContext(buildCtx.Parameters())

	tpl := template.New("eventify").Funcs(map[string]interface{}{
		"package":     p.ctx.PackageName,
		"name":        p.ctx.Name,
		"marshaler":   p.marshaler,
		"unmarshaler": p.unmarshaler,
		"isEvent":     p.isEvent,
	})

	p.tpl = template.Must(tpl.Parse(eventifyTemplate))
}

// Name is the identifier used to identify the module. This value is
// automatically attached to the BuildContext associated with the ModuleBase.
func (p *EventifyPlugin) Name() string { return "eventify" }

// Execute is passed the target files as well as its dependencies in the pkgs
// map.
func (p *EventifyPlugin) Execute(
	targets map[string]pgs.File, pkgs map[string]pgs.Package,
) []pgs.Artifact {
	for _, f := range targets {
		p.generate(f)
	}

	return p.Artifacts()
}

func (p *EventifyPlugin) generate(f pgs.File) {
	var eventCount int
	for _, msg := range f.Messages() {
		if p.isEvent(msg) {
			eventCount++
		}
	}

	if eventCount == 0 {
		return
	}

	name := p.ctx.OutputPath(f).SetExt(".event.go")
	p.AddGeneratorTemplateFile(name.String(), p.tpl, f)
}

func (p *EventifyPlugin) marshaler(m pgs.Message) pgs.Name {
	return p.ctx.Name(m) + "JSONMarshaler"
}

func (p *EventifyPlugin) unmarshaler(m pgs.Message) pgs.Name {
	return p.ctx.Name(m) + "JSONUnmarshaler"
}

func (p *EventifyPlugin) isEvent(m pgs.Message) bool {
	var isEvent bool
	_, err := m.Extension(schema.E_Event, &isEvent)
	p.CheckErr(err, "unable to read event extension from message")

	return isEvent
}

const eventifyTemplate = `package {{ package . }}

import (
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
)

{{ range .AllMessages }}

// EventName returns the name of the event in string.
func (m *{{ name . }}) EventName() string {
	return "{{ name . }}"
}

// {{ marshaler . }} describes the default jsonpb.Marshaler used by all 
// instances of {{ name . }}.
var {{ marshaler . }} = new(protojson.MarshalOptions)

// MarshalJSON satisfies the encoding/json Marshaler interface. This method 
// uses the more correct jsonpb package to correctly marshal the message.
func (m *{{ name . }}) MarshalJSON() ([]byte, error) {
	if m == nil {
		return json.Marshal(nil)
	}

	return {{ marshaler . }}.Marshal(m)
}

var _ json.Marshaler = (*{{ name . }})(nil)

// {{ unmarshaler . }} describes the default jsonpb.Unmarshaler used by all 
// instances of {{ name . }}.
var {{ unmarshaler . }} = new(protojson.UnmarshalOptions)

// UnmarshalJSON satisfies the encoding/json Unmarshaler interface. This method 
// uses the more correct jsonpb package to correctly unmarshal the message.
func (m *{{ name . }}) UnmarshalJSON(b []byte) error {
	return {{ unmarshaler . }}.Unmarshal(b, m)
}

var _ json.Unmarshaler = (*{{ name . }})(nil)

{{ end }}
`
