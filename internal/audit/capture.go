package audit

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
)

// ResponseCaptureWriter proxies responses to downstream and keeps a copy for audit logging.
type ResponseCaptureWriter struct {
	inner        http.ResponseWriter
	statusCode   int
	wroteHeader  bool
	bytesWritten int64
	body         bytes.Buffer
	maxBodyBytes int64
}

func NewResponseCaptureWriter(inner http.ResponseWriter, maxBodyBytes int64) *ResponseCaptureWriter {
	if maxBodyBytes < 0 {
		maxBodyBytes = 0
	}
	return &ResponseCaptureWriter{
		inner:        inner,
		maxBodyBytes: maxBodyBytes,
	}
}

func (w *ResponseCaptureWriter) Header() http.Header {
	if w == nil || w.inner == nil {
		return http.Header{}
	}
	return w.inner.Header()
}

func (w *ResponseCaptureWriter) WriteHeader(statusCode int) {
	if w == nil || w.inner == nil {
		return
	}
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.statusCode = statusCode
	w.inner.WriteHeader(statusCode)
}

func (w *ResponseCaptureWriter) Write(data []byte) (int, error) {
	if w == nil || w.inner == nil {
		return 0, io.ErrClosedPipe
	}
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	w.captureBody(data)
	n, err := w.inner.Write(data)
	w.bytesWritten += int64(n)
	return n, err
}

func (w *ResponseCaptureWriter) captureBody(data []byte) {
	if w == nil || w.maxBodyBytes <= 0 || len(data) == 0 {
		return
	}
	remaining := int(w.maxBodyBytes) - w.body.Len()
	if remaining <= 0 {
		return
	}
	if len(data) > remaining {
		data = data[:remaining]
	}
	_, _ = w.body.Write(data)
}

func (w *ResponseCaptureWriter) StatusCode() int {
	if w == nil {
		return 0
	}
	if w.statusCode != 0 {
		return w.statusCode
	}
	if w.bytesWritten > 0 {
		return http.StatusOK
	}
	return 0
}

func (w *ResponseCaptureWriter) BytesWritten() int64 {
	if w == nil {
		return 0
	}
	return w.bytesWritten
}

func (w *ResponseCaptureWriter) BodyBytes() []byte {
	if w == nil {
		return nil
	}
	out := make([]byte, w.body.Len())
	copy(out, w.body.Bytes())
	return out
}

func (w *ResponseCaptureWriter) Flush() {
	if flusher, ok := w.inner.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *ResponseCaptureWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.inner.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return h.Hijack()
}

func (w *ResponseCaptureWriter) Push(target string, opts *http.PushOptions) error {
	pusher, ok := w.inner.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, opts)
}

// CaptureRequestBody snapshots request body for audit and restores the original body for downstream handlers.
func CaptureRequestBody(req *http.Request, maxBodyBytes int64) ([]byte, error) {
	if req == nil || req.Body == nil || maxBodyBytes == 0 {
		return nil, nil
	}
	if maxBodyBytes < 0 {
		maxBodyBytes = 0
	}

	// Prefer GetBody when available to avoid consuming the original request stream.
	if req.GetBody != nil {
		rc, err := req.GetBody()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		full, err := io.ReadAll(rc)
		if err != nil {
			return nil, err
		}
		return truncateCopy(full, maxBodyBytes), nil
	}

	full, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	_ = req.Body.Close()

	clone := make([]byte, len(full))
	copy(clone, full)
	req.Body = io.NopCloser(bytes.NewReader(clone))
	req.GetBody = func() (io.ReadCloser, error) {
		dup := make([]byte, len(clone))
		copy(dup, clone)
		return io.NopCloser(bytes.NewReader(dup)), nil
	}

	return truncateCopy(clone, maxBodyBytes), nil
}

func truncateCopy(data []byte, maxBodyBytes int64) []byte {
	if maxBodyBytes <= 0 || len(data) == 0 {
		return nil
	}
	if int64(len(data)) <= maxBodyBytes {
		out := make([]byte, len(data))
		copy(out, data)
		return out
	}
	out := make([]byte, maxBodyBytes)
	copy(out, data[:maxBodyBytes])
	return out
}
