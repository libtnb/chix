package binder

import (
	"net/http"
	"strings"
)

type queryBinding struct{}

func (*queryBinding) Name() string {
	return "query"
}

func (b *queryBinding) Bind(r *http.Request, out any, enableSplitting ...bool) error {
	data := make(map[string][]string)
	if len(enableSplitting) == 0 {
		enableSplitting = append(enableSplitting, false)
	}

	for k, v := range r.URL.Query() {
		if err := formatBindData(b.Name(), out, data, k, strings.Join(v, ","), enableSplitting[0], true); err != nil {
			return err
		}
	}

	return parse(b.Name(), out, data)
}
