package renderer

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
)

// Server-sent events
// https://html.spec.whatwg.org/multipage/server-sent-events.html

type SSEvent struct {
	Event string
	Data  io.Reader
	ID    string
	Retry uint
}

func SSEventEncode(writer io.Writer, event SSEvent) error {
	buf := bufio.NewWriter(writer)
	if len(event.Event) > 0 {
		_, err := fmt.Fprintf(buf, "event: %s\n", event.Event)
		if err != nil {
			return err
		}
	}
	if len(event.ID) > 0 {
		_, err := fmt.Fprintf(buf, "id: %s\n", event.ID)
		if err != nil {
			return err
		}
	}
	if event.Retry > 0 {
		_, err := fmt.Fprintf(buf, "retry: %d\n", event.Retry)
		if err != nil {
			return err
		}
	}

	_, _ = buf.WriteString("data: ")
	if _, err := io.Copy(buf, event.Data); err != nil {
		return err
	}
	_, _ = buf.WriteString("\n\n")

	return buf.Flush()
}

func SSEventDecode(reader io.Reader) ([]SSEvent, error) {
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// Strip UTF-8 BOM if present (per SSE spec)
	raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})

	// Split into lines, handling CRLF, LF, and CR
	// Replace CRLF with LF first, then CR with LF
	raw = bytes.ReplaceAll(raw, []byte{'\r', '\n'}, []byte{'\n'})
	raw = bytes.ReplaceAll(raw, []byte{'\r'}, []byte{'\n'})
	lines := bytes.Split(raw, []byte{'\n'})

	var dataLines [][]byte
	var event SSEvent
	var events []SSEvent

	for _, line := range lines {
		if len(line) == 0 {
			// Empty line marks the end of an event
			if len(dataLines) == 0 && event.Event == "" {
				continue
			}

			// Combine data lines according to SSE spec
			// Each data field appends value + newline, but the last newline should be removed
			if len(dataLines) > 0 {
				data := bytes.Join(dataLines, []byte{'\n'})
				event.Data = bytes.NewReader(data)
			}

			// Set default event type if not specified
			if event.Event == "" {
				event.Event = "message"
			}

			events = append(events, event)
			event = SSEvent{}
			dataLines = nil
			continue
		}

		// Ignore comment lines
		if bytes.HasPrefix(line, []byte{':'}) {
			continue
		}

		var field, value []byte
		index := bytes.IndexRune(line, ':')
		if index != -1 {
			field = line[:index]
			value = line[index+1:]
			// Remove optional leading space from value (per SSE spec)
			if len(value) > 0 && value[0] == ' ' {
				value = value[1:]
			}
		} else {
			field = line
			value = []byte{}
		}

		// Process field
		switch string(field) {
		case "event":
			event.Event = string(value)
		case "id":
			// Per SSE spec: if the field value does not contain U+0000 NULL
			if !bytes.Contains(value, []byte{0}) {
				event.ID = string(value)
			}
		case "retry":
			// Only process if field value consists of only ASCII digits
			retry, err := strconv.Atoi(string(value))
			if err == nil && retry >= 0 {
				event.Retry = uint(retry)
			}
		case "data":
			dataLines = append(dataLines, value)
		}
	}

	return events, nil
}
