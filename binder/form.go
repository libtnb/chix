package binder

import (
	"mime/multipart"
	"net/http"
	"strings"
)

type formBinding struct{}

func (*formBinding) Name() string {
	return "form"
}

func (b *formBinding) Bind(r *http.Request, out any, enableSplitting ...bool) error {
	data := make(map[string][]string)
	if len(enableSplitting) == 0 {
		enableSplitting = append(enableSplitting, false)
	}

	if err := r.ParseForm(); err != nil {
		return err
	}

	for k, v := range r.PostForm {
		if err := formatBindData(b.Name(), out, data, k, strings.Join(v, ","), enableSplitting[0], true); err != nil {
			return err
		}
	}

	return parse(b.Name(), out, data)
}

func (b *formBinding) BindMultipart(r *http.Request, out any, size int64, enableSplitting ...bool) error {
	if err := r.ParseMultipartForm(size); err != nil {
		return err
	}
	if len(enableSplitting) == 0 {
		enableSplitting = append(enableSplitting, false)
	}

	data := make(map[string][]string)
	for key, values := range r.MultipartForm.Value {
		if err := formatBindData(b.Name(), out, data, key, values, enableSplitting[0], true); err != nil {
			return err
		}
	}

	files := make(map[string][]*multipart.FileHeader)
	for key, values := range r.MultipartForm.File {
		if err := formatBindData(b.Name(), out, files, key, values, enableSplitting[0], true); err != nil {
			return err
		}
	}

	return parse(b.Name(), out, data, files)
}
