package chix

import (
	"bufio"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"sync"

	"github.com/libtnb/chix/v2/renderer"
)

var renderPool = sync.Pool{
	New: func() any {
		return new(Render)
	},
}

// Render struct
type Render struct {
	w              http.ResponseWriter
	r              *http.Request
	statusCode     int
	statusCodeSent bool
	contentTypeSet bool
}

// NewRender creates a new Render instance.
func NewRender(w http.ResponseWriter, r ...*http.Request) *Render {
	render := renderPool.Get().(*Render)
	render.statusCode = http.StatusOK
	render.w = w
	if len(r) > 0 {
		render.r = r[0]
	}

	return render
}

// ContentType sets the Content-Type header for an HTTP response.
func (r *Render) ContentType(v string) {
	r.contentTypeSet = true
	r.w.Header().Set(HeaderContentType, v)
}

// Status sets the HTTP status code for the response, but does not send it yet.
// This is because once the status is sent, no header can be modified.
func (r *Render) Status(status int) {
	r.statusCode = status
}

// SendStatus is a wrapper for WriteHeader method, will send the status code immediately.
func (r *Render) SendStatus(status int) {
	r.statusCode = status
	r.w.WriteHeader(status)
	r.statusCodeSent = true
}

// writeStatus sends the pending status code, at most once per response.
func (r *Render) writeStatus() {
	if !r.statusCodeSent {
		r.w.WriteHeader(r.statusCode)
		r.statusCodeSent = true
	}
}

// Header sets the provided header key/value pair in the response.
func (r *Render) Header(key, value string) {
	r.w.Header().Set(key, value)
}

// Cookie sets a cookie in the response.
func (r *Render) Cookie(cookie *http.Cookie) {
	http.SetCookie(r.w, cookie)
}

// WithoutCookie deletes a cookie in the response. The optional path must match
// the Path attribute the cookie was set with, it defaults to "/".
func (r *Render) WithoutCookie(name string, path ...string) {
	cookiePath := "/"
	if len(path) > 0 {
		cookiePath = path[0]
	}

	http.SetCookie(r.w, &http.Cookie{
		Name:   name,
		Path:   cookiePath,
		MaxAge: -1,
	})
}

// Redirect replies to the request with a redirect to url, which may be a path
// relative to the request path.
func (r *Render) Redirect(url string) {
	if r.r == nil {
		http.Error(r.w, "chix: Redirect requires passing *http.Request", http.StatusInternalServerError)
		return
	}

	http.Redirect(r.w, r.r, url, http.StatusFound)
}

// RedirectPermanent replies to the request with a redirect to url, which may be
// a path relative to the request path.
func (r *Render) RedirectPermanent(url string) {
	if r.r == nil {
		http.Error(r.w, "chix: RedirectPermanent requires passing *http.Request", http.StatusInternalServerError)
		return
	}

	http.Redirect(r.w, r.r, url, http.StatusMovedPermanently)
}

// PlainText writes a string to the response, setting the Content-Type as
// text/plain if not set.
func (r *Render) PlainText(v string) {
	if !r.contentTypeSet {
		r.w.Header().Set(HeaderContentType, MIMETextPlainCharsetUTF8)
	}
	r.writeStatus()
	_, _ = r.w.Write([]byte(v))
}

// Data writes raw bytes to the response, setting the Content-Type as
// application/octet-stream if not set.
func (r *Render) Data(v []byte) {
	if !r.contentTypeSet {
		r.w.Header().Set(HeaderContentType, MIMEOctetStream)
	}
	r.writeStatus()
	_, _ = r.w.Write(v)
}

// HTML writes a string to the response, setting the Content-Type as text/html
// if not set.
func (r *Render) HTML(v string) {
	if !r.contentTypeSet {
		r.w.Header().Set(HeaderContentType, MIMETextHTMLCharsetUTF8)
	}
	r.writeStatus()
	_, _ = r.w.Write([]byte(v))
}

// JSON marshals 'v' to JSON using JSONMarshal and setting the Content-Type as
// application/json if not set.
func (r *Render) JSON(v any) {
	data, err := JSONMarshal(v)
	if err != nil {
		http.Error(r.w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !r.contentTypeSet {
		r.w.Header().Set(HeaderContentType, MIMEApplicationJSONCharsetUTF8)
	}
	r.writeStatus()
	_, _ = r.w.Write(data)
}

// JSONP marshals 'v' to JSON using JSONMarshal and setting the Content-Type as
// application/javascript if not set.
func (r *Render) JSONP(callback string, v any) {
	data, err := JSONMarshal(v)
	if err != nil {
		http.Error(r.w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !r.contentTypeSet {
		r.w.Header().Set(HeaderContentType, MIMEApplicationJavaScriptCharsetUTF8)
	}
	r.writeStatus()
	_, _ = r.w.Write([]byte(callback + "("))
	_, _ = r.w.Write(data)
	_, _ = r.w.Write([]byte(");"))
}

// XML marshals 'v' to XML using XMLMarshal, setting the Content-Type as
// application/xml if not set. It will automatically prepend a generic XML header
// (see encoding/xml.Header) if one is not found in the first 100 bytes of 'v'.
func (r *Render) XML(v any) {
	data, err := XMLMarshal(v)
	if err != nil {
		http.Error(r.w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !r.contentTypeSet {
		r.w.Header().Set(HeaderContentType, MIMEApplicationXMLCharsetUTF8)
	}
	r.writeStatus()

	// Try to find <?xml header in first 100 bytes (just in case there're some XML comments).
	findHeaderUntil := min(len(data), 100)
	if !bytes.Contains(data[:findHeaderUntil], []byte("<?xml")) {
		// No header found. Print it out first.
		_, _ = r.w.Write([]byte(xml.Header))
	}

	_, _ = r.w.Write(data)
}

// NoContent returns a HTTP 204 "No Content" response.
func (r *Render) NoContent() {
	r.statusCode = http.StatusNoContent
	r.writeStatus()
}

// Stream sends a streaming response and returns a boolean
// indicates "Is client disconnected in middle of stream"
func (r *Render) Stream(step func(w io.Writer) bool) bool {
	if r.r == nil {
		http.Error(r.w, "chix: Stream requires passing *http.Request", http.StatusInternalServerError)
		return false
	}

	r.writeStatus()

	for {
		select {
		case <-r.r.Context().Done():
			return true
		default:
			keepOpen := step(r.w)
			r.Flush()
			if !keepOpen {
				return false
			}
		}
	}
}

// EventStream writes a stream of JSON objects from a channel to the response and setting the
// Content-Type as text/event-stream if not set.
func (r *Render) EventStream(v any) {
	if r.r == nil {
		http.Error(r.w, "chix: EventStream requires passing *http.Request", http.StatusInternalServerError)
		return
	}
	typ := reflect.TypeOf(v)
	if typ == nil || typ.Kind() != reflect.Chan {
		kind := "nil"
		if typ != nil {
			kind = typ.Kind().String()
		}
		http.Error(r.w, "chix: EventStream expects a channel, not "+kind, http.StatusInternalServerError)
		return
	}
	if typ.ChanDir() == reflect.SendDir {
		http.Error(r.w, "chix: EventStream expects a receivable channel", http.StatusInternalServerError)
		return
	}

	if !r.contentTypeSet {
		r.w.Header().Set(HeaderContentType, MIMEEventStreamCharsetUTF8)
	}
	r.w.Header().Set(HeaderCacheControl, "no-cache")

	if r.r.ProtoMajor == 1 {
		// An endpoint MUST NOT generate an HTTP/2 message containing connection-specific header fields.
		// Source: RFC7540
		r.w.Header().Set(HeaderConnection, "keep-alive")
	}

	r.writeStatus()

	ctx := r.r.Context()
	for {
		chosen, recv, ok := reflect.Select([]reflect.SelectCase{
			{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ctx.Done())},
			{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(v)},
		})
		switch chosen {
		case 0: // equivalent to: case <-ctx.Done()
			// Only report server-side timeouts; when the context is canceled
			// the client is usually gone and nobody reads the stream anymore.
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				_, _ = r.w.Write([]byte("event: error\ndata: {\"error\":\"server timeout\"}\n\n"))
			}
			return

		default: // equivalent to: case v, ok := <-stream
			if !ok {
				_, _ = r.w.Write([]byte("event: EOF\n\n"))
				return
			}

			data, err := JSONMarshal(recv.Interface())
			if err != nil {
				msg, _ := JSONMarshal(M{"error": err.Error()})
				_, _ = fmt.Fprintf(r.w, "event: error\ndata: %s\n\n", msg)
				r.Flush()
				continue
			}

			_, _ = fmt.Fprintf(r.w, "event: data\ndata: %s\n\n", data)
			r.Flush()
		}
	}
}

// SSEvent writes a Server-Sent Event to the response and setting the
// Content-Type as text/event-stream if not set.
func (r *Render) SSEvent(event renderer.SSEvent) {
	if r.r == nil {
		http.Error(r.w, "chix: SSEvent requires passing *http.Request", http.StatusInternalServerError)
		return
	}

	if !r.contentTypeSet {
		r.w.Header().Set(HeaderContentType, MIMEEventStreamCharsetUTF8)
	}
	r.w.Header().Set(HeaderCacheControl, "no-cache")

	if r.r.ProtoMajor == 1 {
		// An endpoint MUST NOT generate an HTTP/2 message containing connection-specific header fields.
		// Source: RFC7540
		r.w.Header().Set(HeaderConnection, "keep-alive")
	}

	r.writeStatus()
	_ = renderer.SSEventEncode(r.w, event)
}

// File sends a file to the response.
func (r *Render) File(filepath string) {
	if r.r == nil {
		http.Error(r.w, "chix: File requires passing *http.Request", http.StatusInternalServerError)
		return
	}

	http.ServeFile(r.w, r.r, filepath)
}

// Download sends a file to the response and prompting it to be downloaded
// by setting the Content-Disposition header.
func (r *Render) Download(filepath, filename string) {
	if r.r == nil {
		http.Error(r.w, "chix: Download requires passing *http.Request", http.StatusInternalServerError)
		return
	}
	if isASCII(filename) {
		r.Header(HeaderContentDisposition, `attachment; filename="`+quoteEscape(filename)+`"`)
	} else {
		// RFC 5987 requires percent-encoding; PathEscape keeps spaces as %20
		// (QueryEscape would turn them into "+", which browsers keep literally).
		r.Header(HeaderContentDisposition, `attachment; filename*=UTF-8''`+url.PathEscape(filename))
	}

	http.ServeFile(r.w, r.r, filepath)
}

// Flush sends any buffered data to the response.
func (r *Render) Flush() {
	_ = http.NewResponseController(r.w).Flush()
}

// Hijack takes over the underlying connection, letting the caller manage it.
// It returns an error if the underlying ResponseWriter does not support hijacking.
func (r *Render) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(r.w).Hijack()
}

// Release puts the Render instance back into the pool.
func (r *Render) Release() {
	r.w = nil
	r.r = nil
	r.contentTypeSet = false
	r.statusCodeSent = false
	r.statusCode = http.StatusOK
	renderPool.Put(r)
}
