# chix

This package provides some methods that [go-chi/chi](https://github.com/go-chi/chi) lacks, such as binding and rendering.

A lot of the code in this package comes from [Fiber](https://github.com/gofiber/fiber), the last synchronized version: [ed9595231c08a72f838a1c75389d9dc43665d1b2](https://github.com/gofiber/fiber/commit/ed9595231c08a72f838a1c75389d9dc43665d1b2).

## Install

```bash
go get github.com/libtnb/chix/v2
```

## Migrating from v1

- `chix.JSONEncoder`/`JSONDecoder`/`XMLEncoder`/`XMLDecoder` are replaced by `chix.JSONMarshal`/`JSONUnmarshal`/`XMLMarshal`/`XMLUnmarshal`, which accept any implementation matching the standard `Marshal`/`Unmarshal` signatures (e.g. sonic, go-json).
- Query/form/header binding no longer joins repeated keys with commas: `?a=1&a=2` now binds to `["1", "2"]` instead of `["1,2"]`, and values containing commas are no longer split unless splitting is enabled.
- Repeated keys bound to a scalar (non-slice) field resolve to the **last** value, matching gofiber/schema and Fiber: `?role=user&role=admin` binds `role` as `"admin"`. If duplicate parameters are security-sensitive for you, validate or reject them before binding.
- `Bind.JSON`/`Bind.XML`/`Bind.Body` read at most 32MB of the request body by default; pass a custom size to override. A larger body fails with an `*http.MaxBytesError`, which you can map to HTTP 413.
- `Render.JSON`/`JSONP` no longer write a trailing newline.
- `Render.Hijack` now returns `(net.Conn, *bufio.ReadWriter, error)` directly.
- `Render.WithoutCookie` deletes the cookie with `Path=/` by default and accepts an optional path.
- `Bind.Body` returns `chix.ErrUnsupportedMediaType` for unknown content types, and `Bind.URI` returns `chix.ErrNoRouteContext` instead of panicking when the request was not routed by chi.
- `renderer.SSEventEncode` splits multi-line payloads into one `data:` field per line per the SSE spec, and `Data` may be nil.

## Guides

### Custom Marshal and Unmarshal

Package chix supports custom JSON/XML marshal and unmarshal functions, so you can plug in any drop-in replacement of the standard library:

```go
import (
    "github.com/bytedance/sonic"

    "github.com/libtnb/chix/v2"
)

func init() {
    chix.JSONMarshal = sonic.Marshal
    chix.JSONUnmarshal = sonic.Unmarshal
}
```

### Binding

#### Support Binders

- [Form](binder/form.go)
- [Query](binder/query.go)
- [URI](binder/uri.go)
- [Header](binder/header.go)
- [Cookie](binder/cookie.go)
- [JSON](binder/json.go)
- [XML](binder/xml.go)

#### Binding into a Struct

Chix supports binding request data directly into a struct using [gofiber/schema](https://github.com/gofiber/schema). Here's an example:

```go
// Field names must start with an uppercase letter
type Person struct {
	Name string `json:"name" xml:"name" form:"name"`
	Pass string `json:"pass" xml:"pass" form:"pass"`
}

router.Post("/", func(w http.ResponseWriter, r *http.Request) {
	p := new(Person)
	bind := chix.NewBind(r)
	defer bind.Release()

	if err := bind.Body(p); err != nil {
		return err
	}

	log.Println(p.Name) // Output: john
	log.Println(p.Pass) // Output: doe

	// Additional logic...
})

// Run tests with the following curl commands:

// JSON
curl -X POST -H "Content-Type: application/json" --data "{\"name\":\"john\",\"pass\":\"doe\"}" localhost:3000

// XML
curl -X POST -H "Content-Type: application/xml" --data "<login><name>john</name><pass>doe</pass></login>" localhost:3000

// URL-Encoded Form
curl -X POST -H "Content-Type: application/x-www-form-urlencoded" --data "name=john&pass=doe" localhost:3000

// Multipart Form
curl -X POST -F name=john -F pass=doe http://localhost:3000

// Query Parameters
curl -X POST "http://localhost:3000/?name=john&pass=doe"
```

#### Binding into a Map

Chix allows binding request data into a `map[string]string` or `map[string][]string`. Here's an example:

```go
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	params := make(map[string][]string)
	bind := chix.NewBind(r)
	defer bind.Release()

	if err := bind.Query(params); err != nil {
		return err
	}

	log.Println(params["name"])     // Output: [john]
	log.Println(params["pass"])     // Output: [doe]
	log.Println(params["products"]) // Output: [shoe hat]

	// Additional logic...
})

// Run tests with the following curl command:

curl "http://localhost:3000/?name=john&pass=doe&products=shoe&products=hat"
```

### Render

#### Support Methods

- ContentType
- Status
- Header
- Cookie
- WithoutCookie
- Redirect
- RedirectPermanent
- PlainText
- Data
- HTML
- JSON
- JSONP
- XML
- NoContent
- Stream
- EventStream
- SSEvent
- File
- Download
- Flush
- Hijack
- Release

##### Set Content-Type

```go
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	render := chix.NewRender(w)
	defer render.Release()
	render.ContentType("application/json")
	// Your code...
})
```

##### Set Status Code

```go
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	render := chix.NewRender(w)
	defer render.Release()
	render.Status(http.StatusOK)
	// Your code...
})
```

##### Set Headers

```go
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	render := chix.NewRender(w)
	defer render.Release()
	render.Header("X-Custom-Header", "value")
	// Your code...
})
```

##### Set Cookie

Always set Path explicitly, so the cookie can be reliably removed later:

```go
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	render := chix.NewRender(w)
	defer render.Release()
	render.Cookie(&http.Cookie{
		Name:  "token",
		Value: "your-token",
		Path:  "/",
	})
	// Your code...
})
```

##### Remove Cookie

Browsers only remove a cookie when the deletion Path matches the Path it was
set with (cookies set without a Path get a default path derived from the
request URL, see RFC 6265):

```go
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	render := chix.NewRender(w)
	defer render.Release()
	render.WithoutCookie("token")           // deletes with Path=/
	render.WithoutCookie("token", "/admin") // deletes with a custom path
	// Your code...
})
```

##### Redirect

```go
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	render := chix.NewRender(w, r)
	render.Redirect("/new-location")
})
```

##### Permanent Redirect

```go
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	render := chix.NewRender(w, r)
	render.RedirectPermanent("/new-location")
})
```

##### Render Plain Text

```go
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	render := chix.NewRender(w)
 	defer render.Release()
 	render.PlainText("Hello, World!")
})
```

##### Render Raw Data

```go
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	render := chix.NewRender(w)
	defer render.Release()
	render.Data([]byte("Hello, World!"))
})
```

##### Render HTML

```go
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	render := chix.NewRender(w)
	defer render.Release()
	render.HTML("<h1>Hello, World!</h1>")
})
```

##### Render JSON

```go
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	render := chix.NewRender(w)
	defer render.Release()
	render.JSON(chix.M{
		"hello": "world",
	})
})
```

##### Render JSONP

```go
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	render := chix.NewRender(w)
	defer render.Release()
	render.JSONP("callback", chix.M{
		"hello": "world",
	})
})
```

##### Render XML

```go
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	render := chix.NewRender(w)
	defer render.Release()
 
	type Person struct {
		Name string `xml:"name"`
		Age  int    `xml:"age"`
	}
 
	render.XML(Person{Name: "John", Age: 30})
})
```

##### Send No Content

```go
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	render := chix.NewRender(w)
	defer render.Release()
	render.NoContent()
})
```

##### Stream Response

```go
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	render := chix.NewRender(w, r)
	defer render.Release()
 
	clientDisconnected := render.Stream(func(w io.Writer) bool {
		_, _ = w.Write([]byte("chunk of data\n"))
		time.Sleep(100 * time.Millisecond)
		return true // continue streaming
	})
 
	if clientDisconnected {
		// Handle client disconnect
	}
})
```

##### Event Stream

```go
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	render := chix.NewRender(w, r)
 
	ch := make(chan map[string]string)
 
	// In another goroutine
	go func() {
		ch <- map[string]string{"message": "hello"}
		time.Sleep(time.Second)
		ch <- map[string]string{"message": "world"}
		close(ch)
	}()
 
	render.EventStream(ch)
})
```

##### Server-Sent Event

```go
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	render := chix.NewRender(w, r)

	event := renderer.SSEvent{
		Event: "message",
		Data:  strings.NewReader("Hello, World!"),
		ID:    "1",
		Retry: 3000,
	}

	render.SSEvent(event)
})
```

##### Serve File

```go
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	render := chix.NewRender(w, r)
	render.File("path/to/file.txt")
})
```

##### Download File

```go
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	render := chix.NewRender(w, r)
	render.Download("path/to/file.txt", "download.txt")
})
```

##### Flush Buffer

```go
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	render := chix.NewRender(w)
 
	// Write some data
	w.Data([]byte("Some data"))
 
	// Flush immediately to client
	render.Flush()
 
	// Write more data later...
})
```

##### Hijack Connection

```go
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	render := chix.NewRender(w)
 
	conn, bufrw, err := render.Hijack()
	if err != nil {
		// Hijacking not supported or failed
		return
	}
	defer conn.Close()
 
	// Use the hijacked connection
	bufrw.WriteString("HTTP/1.1 200 OK\r\n\r\n")
	bufrw.WriteString("Hello from hijacked connection")
	bufrw.Flush()
})
```
