package render

func (r *Renderer) RenderExportedData(data any) {
	if r.exporter == nil {
		r.IO.ErrOut.Write([]byte("No exporter available\n"))
		return
	}
	r.exporter.Write(r.IO, data)
}
