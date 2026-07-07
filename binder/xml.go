package binder

import "fmt"

type xmlBinding struct{}

func (*xmlBinding) Name() string {
	return "xml"
}

func (*xmlBinding) Bind(body []byte, unmarshal func(data []byte, v any) error, out any) error {
	if err := unmarshal(body, out); err != nil {
		return fmt.Errorf("bind: %w", err)
	}

	return nil
}
