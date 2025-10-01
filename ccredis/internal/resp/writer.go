package redis

import (
	"io"
)

type Writer struct {
	writer io.Writer
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{writer: w}
}

func (w *Writer) Write(v Value) (int, error) {
	b := v.Marshal()
	n, err := w.writer.Write(b)
	if err != nil {
		return 0, err
	}
	return n, err
}

func syntaxErr() Value { return Value{Type: "error", String: "syntax error"} }
func nullVal() Value   { return Value{Type: "null"} }
func ok() Value        { return Value{Type: "string", String: "OK"} }

func errVal(v string) Value  { return Value{Type: "error", String: v} }
func intVal(v int) Value     { return Value{Type: "integer", Int: v} }
func bulkVal(v string) Value { return Value{Type: "bulk", Bulk: v} }
func strVal(v string) Value  { return Value{Type: "string", String: v} }
