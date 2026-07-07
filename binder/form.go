package binder

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"sync"
)

var (
	dataMapPool = sync.Pool{
		New: func() any {
			return make(map[string][]string, 8)
		},
	}
	formFileMapPool = sync.Pool{
		New: func() any {
			return make(map[string][]*multipart.FileHeader)
		},
	}
)

// Keep oversized maps out of the pool so a rare large bind doesn't get retained
// and reused across subsequent requests.
const maxPoolableDataMapSize = 256

func acquireDataMap() map[string][]string {
	m, ok := dataMapPool.Get().(map[string][]string)
	if !ok {
		m = make(map[string][]string, 8)
	}
	return m
}

func releaseDataMap(m map[string][]string) {
	if len(m) > maxPoolableDataMapSize {
		return
	}

	clear(m)
	dataMapPool.Put(m)
}

func acquireFileHeaderMap() map[string][]*multipart.FileHeader {
	m, ok := formFileMapPool.Get().(map[string][]*multipart.FileHeader)
	if !ok {
		m = make(map[string][]*multipart.FileHeader)
	}
	return m
}

func releaseFileHeaderMap(m map[string][]*multipart.FileHeader) {
	if len(m) > maxPoolableDataMapSize {
		return
	}

	clear(m)
	formFileMapPool.Put(m)
}

type formBinding struct{}

func (*formBinding) Name() string {
	return bindingForm
}

func (b *formBinding) Bind(r *http.Request, out any, enableSplitting ...bool) error {
	if len(enableSplitting) == 0 {
		enableSplitting = append(enableSplitting, false)
	}

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("bind: %w", err)
	}

	data := acquireDataMap()
	defer releaseDataMap(data)

	for k, v := range r.PostForm {
		if err := formatBindData(b.Name(), out, data, k, v, enableSplitting[0], true); err != nil {
			return err
		}
	}

	return parse(b.Name(), out, data)
}

func (b *formBinding) BindMultipart(r *http.Request, out any, size int64, enableSplitting ...bool) error {
	if err := r.ParseMultipartForm(size); err != nil {
		return fmt.Errorf("bind: %w", err)
	}
	if len(enableSplitting) == 0 {
		enableSplitting = append(enableSplitting, false)
	}

	data := acquireDataMap()
	defer releaseDataMap(data)

	for key, values := range r.MultipartForm.Value {
		if err := formatBindData(b.Name(), out, data, key, values, enableSplitting[0], true); err != nil {
			return err
		}
	}

	files := acquireFileHeaderMap()
	defer releaseFileHeaderMap(files)

	for key, values := range r.MultipartForm.File {
		if err := formatBindData(b.Name(), out, files, key, values, enableSplitting[0], true); err != nil {
			return err
		}
	}

	return parse(b.Name(), out, data, files)
}
