package chix_test

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/libtnb/chix/v2"
	"github.com/libtnb/chix/v2/renderer"
)

func TestRender_ContentType(t *testing.T) {
	w := httptest.NewRecorder()
	r := chix.NewRender(w)
	r.ContentType("application/json")
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestRender_Status(t *testing.T) {
	w := httptest.NewRecorder()
	r := chix.NewRender(w)
	r.Status(http.StatusNotFound)
	r.PlainText("404 page not found")
	r.Flush()
	require.Equal(t, http.StatusNotFound, w.Code)
	require.Equal(t, "404 page not found", w.Body.String())
}

func TestRender_SendStatus(t *testing.T) {
	w := httptest.NewRecorder()
	r := chix.NewRender(w)
	r.SendStatus(http.StatusNotFound)
	r.Flush()
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestRender_Header(t *testing.T) {
	w := httptest.NewRecorder()
	r := chix.NewRender(w)
	r.Header("X-Custom-Header", "value")
	require.Equal(t, "value", w.Header().Get("X-Custom-Header"))
}

func TestRender_Cookie(t *testing.T) {
	w := httptest.NewRecorder()
	r := chix.NewRender(w)
	cookie := &http.Cookie{Name: "test", Value: "value"}
	r.Cookie(cookie)
	require.Equal(t, "test=value", w.Header().Get("Set-Cookie"))
}

func TestRender_WithoutCookie(t *testing.T) {
	w := httptest.NewRecorder()
	r := chix.NewRender(w)
	r.WithoutCookie("test")
	require.Equal(t, "test=; Path=/; Max-Age=0", w.Header().Get("Set-Cookie"))
}

func TestRender_WithoutCookieCustomPath(t *testing.T) {
	w := httptest.NewRecorder()
	r := chix.NewRender(w)
	r.WithoutCookie("test", "/admin")
	require.Equal(t, "test=; Path=/admin; Max-Age=0", w.Header().Get("Set-Cookie"))
}

func TestRender_PlainText(t *testing.T) {
	w := httptest.NewRecorder()
	r := chix.NewRender(w)
	r.PlainText("hello")
	require.Equal(t, "hello", w.Body.String())
}

func TestRender_Data(t *testing.T) {
	w := httptest.NewRecorder()
	r := chix.NewRender(w)
	r.Data([]byte("data"))
	require.Equal(t, "data", w.Body.String())
}

func TestRender_HTML(t *testing.T) {
	w := httptest.NewRecorder()
	r := chix.NewRender(w)
	r.HTML("<p>hello</p>")
	require.Equal(t, "<p>hello</p>", w.Body.String())
}

func TestRender_JSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := chix.NewRender(w)
	r.JSON(map[string]string{"key": "value"})
	require.Equal(t, `{"key":"value"}`, w.Body.String())
}

func TestRender_JSONP(t *testing.T) {
	w := httptest.NewRecorder()
	r := chix.NewRender(w)
	r.JSONP("callback", map[string]string{"key": "value"})
	require.Equal(t, `callback({"key":"value"});`, w.Body.String())
}

func TestRender_XML(t *testing.T) {
	w := httptest.NewRecorder()
	r := chix.NewRender(w)

	type KeyValue struct {
		XMLName xml.Name `xml:"map"`
		Key     string   `xml:"key"`
		Value   string   `xml:"value"`
	}

	data := KeyValue{Key: "key", Value: "value"}
	r.XML(data)
	require.Equal(t, xml.Header+`<map><key>key</key><value>value</value></map>`, w.Body.String())
}

func TestRender_NoContent(t *testing.T) {
	w := httptest.NewRecorder()
	r := chix.NewRender(w)
	r.NoContent()
	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestRender_File(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	f, err := os.CreateTemp("", "test.txt")
	require.NoError(t, err)
	defer func(name string) {
		_ = os.Remove(name)
	}(f.Name())
	_, err = f.WriteString("test file content")
	require.NoError(t, err)
	r := chix.NewRender(w, req)
	r.File(f.Name())
	require.Equal(t, "test file content", w.Body.String())
}

func TestRender_Download(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	f, err := os.CreateTemp("", "test.txt")
	require.NoError(t, err)
	defer func(name string) {
		_ = os.Remove(name)
	}(f.Name())
	_, err = f.WriteString("test file content")
	require.NoError(t, err)
	r := chix.NewRender(w, req)
	r.Download(f.Name(), "test.txt")
	require.Equal(t, `attachment; filename="test.txt"`, w.Header().Get("Content-Disposition"))
	require.Equal(t, "test file content", w.Body.String())
}

func TestRender_Redirect(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	r := chix.NewRender(w, req)
	r.Redirect("/new-location")
	require.Equal(t, http.StatusFound, w.Code)
	require.Equal(t, "/new-location", w.Header().Get("Location"))
}

func TestRender_RedirectWithoutRequest(t *testing.T) {
	w := httptest.NewRecorder()
	r := chix.NewRender(w)
	r.Redirect("/new-location")
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "Redirect requires passing *http.Request")
}

func TestRender_RedirectPermanent(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	r := chix.NewRender(w, req)
	r.RedirectPermanent("/new-location")
	require.Equal(t, http.StatusMovedPermanently, w.Code)
	require.Equal(t, "/new-location", w.Header().Get("Location"))
}

func TestRender_RedirectPermanentWithoutRequest(t *testing.T) {
	w := httptest.NewRecorder()
	r := chix.NewRender(w)
	r.RedirectPermanent("/new-location")
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "RedirectPermanent requires passing *http.Request")
}

func TestRender_Stream(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	r := chix.NewRender(w, req)

	count := 0
	clientDisconnected := r.Stream(func(w io.Writer) bool {
		if count >= 3 {
			return false
		}
		_, _ = fmt.Fprintf(w, "chunk %d\n", count)
		count++
		return true
	})

	require.False(t, clientDisconnected)
	require.Equal(t, "chunk 0\nchunk 1\nchunk 2\n", w.Body.String())
}

func TestRender_StreamWithoutRequest(t *testing.T) {
	w := httptest.NewRecorder()
	r := chix.NewRender(w)
	r.Stream(func(w io.Writer) bool { return false })
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "Stream requires passing *http.Request")
}

func TestRender_EventStream(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	r := chix.NewRender(w, req)

	ch := make(chan map[string]string, 2)
	ch <- map[string]string{"message": "hello"}
	ch <- map[string]string{"message": "world"}
	close(ch)

	r.EventStream(ch)

	require.Equal(t, "text/event-stream; charset=utf-8", w.Header().Get("Content-Type"))
	require.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	if req.ProtoMajor == 1 {
		require.Equal(t, "keep-alive", w.Header().Get("Connection"))
	}

	response := w.Body.String()
	require.Contains(t, response, `event: data`)
	require.Contains(t, response, `data: {"message":"hello"}`)
	require.Contains(t, response, `data: {"message":"world"}`)
	require.Contains(t, response, `event: EOF`)
}

func TestRender_EventStreamWithoutRequest(t *testing.T) {
	w := httptest.NewRecorder()
	r := chix.NewRender(w)
	ch := make(chan map[string]string)
	close(ch)
	r.EventStream(ch)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "EventStream requires passing *http.Request")
}

func TestRender_EventStreamWithNonChannel(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	r := chix.NewRender(w, req)
	r.EventStream("not-a-channel")
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "EventStream expects a channel")
}

func TestRender_SSEvent(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	r := chix.NewRender(w, req)

	event := renderer.SSEvent{
		Event: "message",
		Data:  strings.NewReader("hello world"),
		ID:    "123",
		Retry: 3000,
	}

	r.SSEvent(event)

	require.Equal(t, "text/event-stream; charset=utf-8", w.Header().Get("Content-Type"))
	require.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	if req.ProtoMajor == 1 {
		require.Equal(t, "keep-alive", w.Header().Get("Connection"))
	}

	response := w.Body.String()
	require.Contains(t, response, "event: message\n")
	require.Contains(t, response, "data: hello world\n")
	require.Contains(t, response, "id: 123\n")
	require.Contains(t, response, "retry: 3000\n")
}

func TestRender_SSEventWithoutRequest(t *testing.T) {
	w := httptest.NewRecorder()
	r := chix.NewRender(w)

	event := renderer.SSEvent{
		Event: "message",
		Data:  strings.NewReader("hello world"),
	}

	r.SSEvent(event)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "SSEvent requires passing *http.Request")
}

func TestRender_Flush(t *testing.T) {
	// Create a custom ResponseWriter that implements http.Flusher
	flushCalled := false
	w := &mockFlusher{
		ResponseRecorder: httptest.NewRecorder(),
		flushFn: func() {
			flushCalled = true
		},
	}

	r := chix.NewRender(w)
	r.Flush()
	require.True(t, flushCalled)
}

func TestRender_Hijack(t *testing.T) {
	// Create a custom ResponseWriter that implements http.Hijacker
	w := &mockHijacker{
		ResponseRecorder: httptest.NewRecorder(),
	}

	r := chix.NewRender(w)
	conn, bufrw, err := r.Hijack()
	require.NoError(t, err)
	require.Nil(t, conn)
	require.NotNil(t, bufrw)
	_, err = bufrw.WriteString("hello world")
	require.NoError(t, err)
	err = bufrw.Flush()
	require.NoError(t, err)
	require.Equal(t, "hello world", w.Body.String())
}

func TestRender_HijackNotSupported(t *testing.T) {
	w := httptest.NewRecorder()
	r := chix.NewRender(w)
	_, _, err := r.Hijack()
	require.Error(t, err)
}

func TestRender_SSEventMultipleCalls(t *testing.T) {
	// SSEvent is meant to be called repeatedly on one stream; the status code
	// must be written exactly once or net/http logs "superfluous WriteHeader".
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(os.Stderr)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r := chix.NewRender(w, req)
		defer r.Release()
		r.SSEvent(renderer.SSEvent{Event: "a", Data: strings.NewReader("1")})
		r.SSEvent(renderer.SSEvent{Event: "b", Data: strings.NewReader("2")})
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	require.Contains(t, string(body), "event: a\ndata: 1\n\n")
	require.Contains(t, string(body), "event: b\ndata: 2\n\n")
	require.NotContains(t, logBuf.String(), "superfluous")
}

func TestRender_EventStreamWithNil(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	r := chix.NewRender(w, req)
	r.EventStream(nil)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "EventStream expects a channel")
}

func TestRender_EventStreamWithSendOnlyChannel(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	r := chix.NewRender(w, req)
	ch := make(chan string)
	r.EventStream((chan<- string)(ch))
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "receivable channel")
}

func TestRender_DownloadNonASCIIFilename(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	f, err := os.CreateTemp("", "test.txt")
	require.NoError(t, err)
	defer func(name string) {
		_ = os.Remove(name)
	}(f.Name())
	_, err = f.WriteString("test file content")
	require.NoError(t, err)
	r := chix.NewRender(w, req)
	// Spaces must be percent-encoded (%20), not "+", per RFC 5987.
	r.Download(f.Name(), "报告 final.pdf")
	require.Equal(t, `attachment; filename*=UTF-8''%E6%8A%A5%E5%91%8A%20final.pdf`, w.Header().Get("Content-Disposition"))
}

func TestRender_Release(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	r := chix.NewRender(w, req)
	r.ContentType("application/json")

	// Should be able to release and reuse
	r.Release()

	// After release, verify we can use a new render instance
	w2 := httptest.NewRecorder()
	r2 := chix.NewRender(w2)
	r2.HTML("<p>test</p>")
	require.Equal(t, "text/html; charset=utf-8", w2.Header().Get("Content-Type"))
}

// Mock types for testing
type mockFlusher struct {
	*httptest.ResponseRecorder
	flushFn func()
}

func (m *mockFlusher) Flush() {
	if m.flushFn != nil {
		m.flushFn()
	}
}

type mockHijacker struct {
	*httptest.ResponseRecorder
}

func (m *mockHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, bufio.NewReadWriter(bufio.NewReader(m.Body), bufio.NewWriter(m.Body)), nil
}
