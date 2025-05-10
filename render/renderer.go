package render

import (
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/iostreams"
)

type Renderer struct {
	IO       *iostreams.IOStreams
	Color    bool
	exporter cmdutil.Exporter
}

func NewRenderer(ex cmdutil.Exporter) *Renderer {
	return &Renderer{
		IO:       iostreams.System(),
		exporter: ex,
	}
}

func (r *Renderer) SetColor(colorFlag string) {
	if colorFlag == "always" {
		r.Color = true
	} else if colorFlag == "never" {
		r.Color = false
	} else {
		r.Color = r.IO.ColorEnabled()
	}
}

func ToString(v any) string {
	if v, ok := v.(*any); ok {
		if v == nil {
			return "nil"
		}
		return toString(*v)
	}
	return toString(v)
}

func toString(v any) string {
	if str, ok := v.(string); ok {
		return str
	} else if str, ok := v.(*string); ok {
		if str != nil {
			return *str
		}
	} else if b, ok := v.(bool); ok {
		if b {
			return "YES"
		}
		return "NO"
	}
	return ""
}
