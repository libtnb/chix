package binder

import (
	"net/http"
)

type queryBinding struct{}

func (*queryBinding) Name() string {
	return bindingQuery
}

func (b *queryBinding) Bind(r *http.Request, out any, enableSplitting ...bool) error {
	if len(enableSplitting) == 0 {
		enableSplitting = append(enableSplitting, false)
	}

	data := acquireDataMap()
	defer releaseDataMap(data)

	for k, v := range r.URL.Query() {
		if err := formatBindData(b.Name(), out, data, k, v, enableSplitting[0], true); err != nil {
			return err
		}
	}

	return parse(b.Name(), out, data)
}
