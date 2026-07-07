package binder

import (
	"net/http"
)

type cookieBinding struct{}

func (*cookieBinding) Name() string {
	return bindingCookie
}

func (b *cookieBinding) Bind(r *http.Request, out any, enableSplitting ...bool) error {
	if len(enableSplitting) == 0 {
		enableSplitting = append(enableSplitting, false)
	}

	data := acquireDataMap()
	defer releaseDataMap(data)

	for _, cookie := range r.Cookies() {
		k := cookie.Name
		v := cookie.Value

		if err := formatBindData(b.Name(), out, data, k, v, enableSplitting[0], true); err != nil {
			return err
		}
	}

	return parse(b.Name(), out, data)
}
