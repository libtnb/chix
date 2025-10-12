# Chix

This package provides some methods that Chi lacks, such as binding and rendering.

A lot of the code in this package comes from [Fiber](https://github.com/gofiber/fiber), the last synchronized version: [ed9595231c08a72f838a1c75389d9dc43665d1b2](https://github.com/gofiber/fiber/commit/ed9595231c08a72f838a1c75389d9dc43665d1b2).

## Guides

### Custom Encoders and Decoders

Chix supports custom JSON/XML encoders and decoders. Here's an example:

```go
import (
    "encoding/json"
    "encoding/xml"

    "github.com/libtnb/chix"
)

func init() {
    chix.JSONEncoder = json.NewEncoder
    chix.JSONDecoder = json.NewDecoder
    chix.XMLEncoder = xml.NewEncoder
    chix.XMLDecoder = xml.NewDecoder
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

```go
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	render := chix.NewRender(w)
	defer render.Release()
	render.Cookie(&http.Cookie{
		Name:  "token",
		Value: "your-token",
	})
	// Your code...
})
```

##### Remove Cookie

```go
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	render := chix.NewRender(w)
	defer render.Release()
	render.WithoutCookie("token")
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
 
	hijacker, ok := render.Hijack()
	if !ok {
		// Hijacking not supported
		return
	}
 
	conn, bufrw, err := hijacker.Hijack()
	if err != nil {
		// Handle error
		return
	}
	defer conn.Close()
 
	// Use the hijacked connection
	bufrw.WriteString("HTTP/1.1 200 OK\r\n\r\n")
	bufrw.WriteString("Hello from hijacked connection")
	bufrw.Flush()
})
```
