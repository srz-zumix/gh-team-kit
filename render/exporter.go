package render

import "fmt"

func (r *Renderer) renderExportedData(data any) error {
	if r.exporter == nil {
		return fmt.Errorf("no exporter available")
	}
	return r.exporter.Write(r.IO, data)
}

func (r *Renderer) RenderExportedData(data any) {
	err := r.renderExportedData(data)
	if err != nil {
		fmt.Fprintf(r.IO.ErrOut, "Error rendering exported data: %v\n", err)
	}
}
