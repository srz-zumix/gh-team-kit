package render

import "fmt"

func (r *Renderer) RenderExportedData(data any) {
	if r.exporter == nil {
		defer fmt.Fprintln(r.IO.ErrOut, "No exporter available")
		return
	}
	defer r.exporter.Write(r.IO, data)
}
