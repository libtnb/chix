package binder

import (
	"net/http"
)

type headerBinding struct{}

func (*headerBinding) Name() string {
	return bindingHeader
}

func (b *headerBinding) Bind(r *http.Request, out any, enableSplitting ...bool) error {
	if len(enableSplitting) == 0 {
		enableSplitting = append(enableSplitting, false)
	}

	data := acquireDataMap()
	defer releaseDataMap(data)

	for k, v := range r.Header {
		if err := formatBindData(b.Name(), out, data, k, v, enableSplitting[0], false); err != nil {
			return err
		}
	}

	return parse(b.Name(), out, data)
}
