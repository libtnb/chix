package binder

import (
	"net/http"
	"strings"
)

type headerBinding struct{}

func (*headerBinding) Name() string {
	return "header"
}

func (b *headerBinding) Bind(r *http.Request, out any, enableSplitting ...bool) error {
	data := make(map[string][]string)
	if len(enableSplitting) == 0 {
		enableSplitting = append(enableSplitting, false)
	}

	for k, v := range r.Header {
		if err := formatBindData(b.Name(), out, data, k, strings.Join(v, ","), enableSplitting[0], false); err != nil {
			return err
		}
	}

	return parse(b.Name(), out, data)
}
