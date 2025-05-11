package render

import (
	"fmt"

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
	switch colorFlag {
	case "always":
		r.Color = true
	case "never":
		r.Color = false
	default:
		r.Color = r.IO.ColorEnabled()
	}
}

func (r *Renderer) WriteLine(line string) {
	_, err := fmt.Fprintln(r.IO.Out, line)
	if err != nil {
		r.WriteError(err)
	}
}

func (r *Renderer) WriteError(err error) {
	fmt.Fprintf(r.IO.ErrOut, "%v\n", err) // nolint
}

func ToString(v any) string {
	if str, ok := v.(*string); ok {
		if str == nil {
			return ""
		}
		return *str
	} else if b, ok := v.(*bool); ok {
		if b == nil {
			return ""
		}
		return toString(*b)
	} else if i, ok := v.(*int); ok {
		if i == nil {
			return ""
		}
		return toString(*i)
	} else if i, ok := v.(*int64); ok {
		if i == nil {
			return ""
		}
		return toString(*i)
	} else if i, ok := v.(*int32); ok {
		if i == nil {
			return ""
		}
		return toString(*i)
	} else if i, ok := v.(*int8); ok {
		if i == nil {
			return ""
		}
		return toString(*i)
	} else if i, ok := v.(*int16); ok {
		if i == nil {
			return ""
		}
		return toString(*i)
	} else if i, ok := v.(*uint); ok {
		if i == nil {
			return ""
		}
		return toString(*i)
	} else if i, ok := v.(*uint64); ok {
		if i == nil {
			return ""
		}
		return toString(*i)
	} else if i, ok := v.(*uint32); ok {
		if i == nil {
			return ""
		}
		return toString(*i)
	} else if i, ok := v.(*uint8); ok {
		if i == nil {
			return ""
		}
		return toString(*i)
	} else if i, ok := v.(*uint16); ok {
		if i == nil {
			return ""
		}
		return toString(*i)
	} else if i, ok := v.(*float64); ok {
		if i == nil {
			return ""
		}
		return toString(*i)
	} else if i, ok := v.(*float32); ok {
		if i == nil {
			return ""
		}
		return toString(*i)
	} else if e, ok := v.(*error); ok {
		if e == nil {
			return ""
		}
		return toString(*e)
	}
	return toString(v)
}

func toString(v any) string {
	if str, ok := v.(string); ok {
		return str
	} else if b, ok := v.(bool); ok {
		if b {
			return "YES"
		}
		return "NO"
	} else if i, ok := v.(int); ok {
		return fmt.Sprintf("%d", i)
	} else if i, ok := v.(int64); ok {
		return fmt.Sprintf("%d", i)
	} else if i, ok := v.(int32); ok {
		return fmt.Sprintf("%d", i)
	} else if i, ok := v.(int8); ok {
		return fmt.Sprintf("%d", i)
	} else if i, ok := v.(int16); ok {
		return fmt.Sprintf("%d", i)
	} else if i, ok := v.(uint); ok {
		return fmt.Sprintf("%d", i)
	} else if i, ok := v.(uint64); ok {
		return fmt.Sprintf("%d", i)
	} else if i, ok := v.(uint32); ok {
		return fmt.Sprintf("%d", i)
	} else if i, ok := v.(uint8); ok {
		return fmt.Sprintf("%d", i)
	} else if i, ok := v.(uint16); ok {
		return fmt.Sprintf("%d", i)
	} else if f, ok := v.(float64); ok {
		return fmt.Sprintf("%f", f)
	} else if f, ok := v.(float32); ok {
		return fmt.Sprintf("%f", f)
	} else if i, ok := v.(error); ok {
		return i.Error()
	}
	return ""
}
