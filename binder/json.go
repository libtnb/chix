package binder

import "fmt"

type jsonBinding struct{}

func (*jsonBinding) Name() string {
	return "json"
}

func (*jsonBinding) Bind(body []byte, unmarshal func(data []byte, v any) error, out any) error {
	if err := unmarshal(body, out); err != nil {
		return fmt.Errorf("bind: %w", err)
	}

	return nil
}
