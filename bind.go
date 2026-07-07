package chix

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/libtnb/chix/v2/binder"
)

// Bind errors
var (
	// ErrUnsupportedMediaType is returned by Body when no binder matches the request's Content-Type.
	ErrUnsupportedMediaType = binder.ErrSuitableContentNotFound
	// ErrNoRouteContext is returned by URI when the request carries no chi route context.
	ErrNoRouteContext = errors.New("chix: no chi route context in request")
)

// defaultBodyLimit is the default maximum memory in bytes used to read and
// parse a request body. net/http, unlike fasthttp, enforces no body size
// limit on its own.
const defaultBodyLimit = 32 << 20 // 32MB

var bindPool = sync.Pool{
	New: func() any {
		return new(Bind)
	},
}

// Bind struct
type Bind struct {
	r               *http.Request
	enableSplitting bool
}

// NewBind creates a new Bind instance.
func NewBind(r *http.Request, enableSplitting ...bool) *Bind {
	b := bindPool.Get().(*Bind)
	b.r = r
	if len(enableSplitting) > 0 {
		b.enableSplitting = enableSplitting[0]
	}

	return b
}

// Header binds the request header strings into the struct, map[string]string and map[string][]string.
func (b *Bind) Header(out any) error {
	return binder.HeaderBinder.Bind(b.r, out, b.enableSplitting)
}

// Cookie binds the request cookie strings into the struct, map[string]string and map[string][]string.
// NOTE: If your cookie is like key=val1,val2; they'll be binded as an slice if your map is map[string][]string. Else, it'll use last element of cookie.
func (b *Bind) Cookie(out any) error {
	return binder.CookieBinder.Bind(b.r, out, b.enableSplitting)
}

// Query binds the query string into the struct, map[string]string and map[string][]string.
func (b *Bind) Query(out any) error {
	return binder.QueryBinder.Bind(b.r, out, b.enableSplitting)
}

// readBody reads the request body up to the given size limit, default is 32MB.
// A larger body fails with an *http.MaxBytesError.
func (b *Bind) readBody(size ...int64) ([]byte, error) {
	limit := int64(defaultBodyLimit)
	if len(size) > 0 {
		limit = size[0]
	}

	body, err := io.ReadAll(http.MaxBytesReader(nil, b.r.Body, limit))
	if err != nil {
		return nil, fmt.Errorf("bind: %w", err)
	}

	return body, nil
}

// JSON binds the body string into the struct using JSONUnmarshal.
// Parameter size is the maximum body size in bytes to read, default is 32MB;
// a larger body fails with an *http.MaxBytesError.
func (b *Bind) JSON(out any, size ...int64) error {
	body, err := b.readBody(size...)
	if err != nil {
		return err
	}

	return binder.JSONBinder.Bind(body, JSONUnmarshal, out)
}

// XML binds the body string into the struct using XMLUnmarshal.
// Parameter size is the maximum body size in bytes to read, default is 32MB;
// a larger body fails with an *http.MaxBytesError.
func (b *Bind) XML(out any, size ...int64) error {
	body, err := b.readBody(size...)
	if err != nil {
		return err
	}

	return binder.XMLBinder.Bind(body, XMLUnmarshal, out)
}

// Form binds the form into the struct, map[string]string and map[string][]string.
func (b *Bind) Form(out any) error {
	return binder.FormBinder.Bind(b.r, out, b.enableSplitting)
}

// URI binds the route parameters into the struct, map[string]string and map[string][]string.
// It returns ErrNoRouteContext if the request was not routed by chi.
func (b *Bind) URI(out any) error {
	ctx := chi.RouteContext(b.r.Context())
	if ctx == nil {
		return ErrNoRouteContext
	}

	return binder.URIBinder.Bind(ctx.URLParams.Keys, ctx.URLParam, out)
}

// MultipartForm binds the multipart form into the struct, map[string]string and map[string][]string.
// Parameter size is the maximum memory in bytes used to parse the form, default is 32MB.
func (b *Bind) MultipartForm(out any, size ...int64) error {
	if len(size) == 0 {
		size = append(size, defaultBodyLimit)
	}

	return binder.FormBinder.BindMultipart(b.r, out, size[0], b.enableSplitting)
}

// Body binds the request body into the struct, map[string]string and map[string][]string.
// It supports decoding the following content types based on the Content-Type header:
// application/json, application/xml, application/x-www-form-urlencoded, multipart/form-data.
// The optional size parameter is the maximum memory in bytes used to read and parse
// the body, default is 32MB; a larger JSON/XML body fails with an *http.MaxBytesError.
// If no supported mime type of body is matched, it returns ErrUnsupportedMediaType.
func (b *Bind) Body(out any, size ...int64) error {
	// Get content-type
	ctype := strings.ToLower(b.r.Header.Get("Content-Type"))
	ctype = binder.FilterFlags(parseVendorSpecificContentType(ctype))

	// Parse body accordingly
	switch ctype {
	case MIMEApplicationJSON:
		return b.JSON(out, size...)
	case MIMETextXML, MIMEApplicationXML:
		return b.XML(out, size...)
	case MIMEApplicationForm:
		return b.Form(out)
	case MIMEMultipartForm:
		return b.MultipartForm(out, size...)
	}

	// No suitable content type found
	return ErrUnsupportedMediaType
}

// Release releases the Bind instance back into the pool.
func (b *Bind) Release() {
	b.r = nil
	b.enableSplitting = false
	bindPool.Put(b)
}
