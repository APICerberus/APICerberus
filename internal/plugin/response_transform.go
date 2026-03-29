package plugin

import (
	"bytes"
	"net/http"
	"strconv"
)

// ResponseTransformConfig configures response mutations after proxying.
type ResponseTransformConfig struct {
	AddHeaders    map[string]string
	RemoveHeaders []string
	ReplaceBody   string
}

// ResponseTransform modifies captured upstream responses before writing to client.
type ResponseTransform struct {
	addHeaders    map[string]string
	removeHeaders []string
	replaceBody   *string
}

func NewResponseTransform(cfg ResponseTransformConfig) *ResponseTransform {
	addHeaders := normalizeHeaderMap(cfg.AddHeaders)
	removeHeaders := normalizeHeaderList(cfg.RemoveHeaders)

	var replaceBody *string
	if cfg.ReplaceBody != "" {
		value := cfg.ReplaceBody
		replaceBody = &value
	}

	return &ResponseTransform{
		addHeaders:    addHeaders,
		removeHeaders: removeHeaders,
		replaceBody:   replaceBody,
	}
}

func (t *ResponseTransform) Name() string  { return "response-transform" }
func (t *ResponseTransform) Phase() Phase  { return PhasePostProxy }
func (t *ResponseTransform) Priority() int { return 40 }

// Apply wraps response writer to capture headers/body for post-proxy mutations.
func (t *ResponseTransform) Apply(in *PipelineContext) {
	if t == nil || in == nil || in.ResponseWriter == nil {
		return
	}
	if _, exists := in.ResponseWriter.(*CaptureResponseWriter); exists {
		return
	}
	in.ResponseWriter = NewCaptureResponseWriter(in.ResponseWriter)
}

// AfterProxy mutates captured response and flushes it to the original writer.
func (t *ResponseTransform) AfterProxy(in *PipelineContext, _ error) {
	if t == nil || in == nil {
		return
	}
	capture, ok := in.ResponseWriter.(*CaptureResponseWriter)
	if !ok || !capture.HasCaptured() {
		return
	}

	for _, key := range t.removeHeaders {
		capture.Header().Del(key)
	}
	for key, value := range t.addHeaders {
		capture.Header().Set(key, value)
	}
	if t.replaceBody != nil {
		capture.SetBody([]byte(*t.replaceBody))
	}
}

// CaptureResponseWriter buffers status/headers/body until Flush is called.
type CaptureResponseWriter struct {
	inner       http.ResponseWriter
	header      http.Header
	status      int
	wroteHeader bool
	body        bytes.Buffer
	flushed     bool
}

func NewCaptureResponseWriter(inner http.ResponseWriter) *CaptureResponseWriter {
	captured := make(http.Header)
	if inner != nil {
		for key, values := range inner.Header() {
			for _, value := range values {
				captured.Add(key, value)
			}
		}
	}
	return &CaptureResponseWriter{
		inner:  inner,
		header: captured,
	}
}

func (w *CaptureResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *CaptureResponseWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.body.Write(data)
}

func (w *CaptureResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.status = statusCode
}

func (w *CaptureResponseWriter) HasCaptured() bool {
	if w == nil {
		return false
	}
	return w.wroteHeader || w.body.Len() > 0
}

func (w *CaptureResponseWriter) SetBody(data []byte) {
	if w == nil {
		return
	}
	w.body.Reset()
	_, _ = w.body.Write(data)
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
}

func (w *CaptureResponseWriter) Flush() error {
	if w == nil || w.inner == nil || w.flushed {
		return nil
	}
	status := w.status
	if status <= 0 {
		status = http.StatusOK
	}

	dst := w.inner.Header()
	for key := range dst {
		delete(dst, key)
	}
	for key, values := range w.Header() {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
	w.inner.WriteHeader(status)
	_, err := w.inner.Write(w.body.Bytes())
	w.flushed = true
	return err
}

func (w *CaptureResponseWriter) BodyBytes() []byte {
	if w == nil {
		return nil
	}
	out := make([]byte, w.body.Len())
	copy(out, w.body.Bytes())
	return out
}

func (w *CaptureResponseWriter) IsFlushed() bool {
	if w == nil {
		return false
	}
	return w.flushed
}
