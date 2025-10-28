package renderer

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSSEventEncode_FullEvent(t *testing.T) {
	var buf bytes.Buffer
	event := SSEvent{
		Event: "message",
		Data:  strings.NewReader("hello world"),
		ID:    "123",
		Retry: 3000,
	}

	err := SSEventEncode(&buf, event)
	require.NoError(t, err)

	result := buf.String()
	require.Contains(t, result, "event: message\n")
	require.Contains(t, result, "id: 123\n")
	require.Contains(t, result, "retry: 3000\n")
	require.Contains(t, result, "data: hello world\n\n")
}

func TestSSEventEncode_MinimalEvent(t *testing.T) {
	var buf bytes.Buffer
	event := SSEvent{
		Data: strings.NewReader("test data"),
	}

	err := SSEventEncode(&buf, event)
	require.NoError(t, err)

	result := buf.String()
	require.Equal(t, "data: test data\n\n", result)
}

func TestSSEventEncode_OnlyEventName(t *testing.T) {
	var buf bytes.Buffer
	event := SSEvent{
		Event: "custom-event",
		Data:  strings.NewReader("some data"),
	}

	err := SSEventEncode(&buf, event)
	require.NoError(t, err)

	result := buf.String()
	require.Contains(t, result, "event: custom-event\n")
	require.Contains(t, result, "data: some data\n\n")
}

func TestSSEventEncode_WithID(t *testing.T) {
	var buf bytes.Buffer
	event := SSEvent{
		Data: strings.NewReader("data with id"),
		ID:   "event-456",
	}

	err := SSEventEncode(&buf, event)
	require.NoError(t, err)

	result := buf.String()
	require.Contains(t, result, "id: event-456\n")
	require.Contains(t, result, "data: data with id\n\n")
}

func TestSSEventEncode_WithRetry(t *testing.T) {
	var buf bytes.Buffer
	event := SSEvent{
		Data:  strings.NewReader("retry data"),
		Retry: 5000,
	}

	err := SSEventEncode(&buf, event)
	require.NoError(t, err)

	result := buf.String()
	require.Contains(t, result, "retry: 5000\n")
	require.Contains(t, result, "data: retry data\n\n")
}

func TestSSEventEncode_MultilineData(t *testing.T) {
	var buf bytes.Buffer
	event := SSEvent{
		Event: "multiline",
		Data:  strings.NewReader("line1\nline2\nline3"),
	}

	err := SSEventEncode(&buf, event)
	require.NoError(t, err)

	result := buf.String()
	require.Contains(t, result, "event: multiline\n")
	require.Contains(t, result, "data: line1\nline2\nline3\n\n")
}

func TestSSEventDecode_SingleEvent(t *testing.T) {
	input := "event: message\nid: 123\nretry: 3000\ndata: hello world\n\n"
	events, err := SSEventDecode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "message", event.Event)
	require.Equal(t, "123", event.ID)
	require.Equal(t, uint(3000), event.Retry)

	data, err := io.ReadAll(event.Data)
	require.NoError(t, err)
	require.Equal(t, "hello world", string(data))
}

func TestSSEventDecode_MultipleEvents(t *testing.T) {
	input := "event: msg1\ndata: first\n\nevent: msg2\ndata: second\n\n"
	events, err := SSEventDecode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 2)

	require.Equal(t, "msg1", events[0].Event)
	data1, err := io.ReadAll(events[0].Data)
	require.NoError(t, err)
	require.Equal(t, "first", string(data1))

	require.Equal(t, "msg2", events[1].Event)
	data2, err := io.ReadAll(events[1].Data)
	require.NoError(t, err)
	require.Equal(t, "second", string(data2))
}

func TestSSEventDecode_DefaultEventType(t *testing.T) {
	input := "data: no event type specified\n\n"
	events, err := SSEventDecode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "message", events[0].Event)
}

func TestSSEventDecode_MultilineData(t *testing.T) {
	input := "event: multiline\ndata: line1\ndata: line2\ndata:line3\n\n"
	events, err := SSEventDecode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)

	data, err := io.ReadAll(events[0].Data)
	require.NoError(t, err)
	require.Equal(t, "line1\nline2\nline3", string(data))
}

func TestSSEventDecode_CommentsIgnored(t *testing.T) {
	input := ": this is a comment\nevent: test\ndata: actual data\n\n"
	events, err := SSEventDecode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "test", events[0].Event)
}

func TestSSEventDecode_EmptyLines(t *testing.T) {
	input := "\n\nevent: test\ndata: data\n\n\n"
	events, err := SSEventDecode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "test", events[0].Event)
}

func TestSSEventDecode_OnlyData(t *testing.T) {
	input := "data: simple message\n\n"
	events, err := SSEventDecode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)

	data, err := io.ReadAll(events[0].Data)
	require.NoError(t, err)
	require.Equal(t, "simple message", string(data))
}

func TestSSEventDecode_InvalidRetry(t *testing.T) {
	input := "retry: invalid\ndata: test\n\n"
	events, err := SSEventDecode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, uint(0), events[0].Retry)
}

func TestSSEventDecode_FieldWithoutColon(t *testing.T) {
	input := "event\ndata: test\n\n"
	events, err := SSEventDecode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
}

func TestSSEventEncodeDecode_RoundTrip(t *testing.T) {
	original := SSEvent{
		Event: "test-event",
		Data:  strings.NewReader("test data"),
		ID:    "test-id",
		Retry: 1000,
	}

	var buf bytes.Buffer
	err := SSEventEncode(&buf, original)
	require.NoError(t, err)

	events, err := SSEventDecode(&buf)
	require.NoError(t, err)
	require.Len(t, events, 1)

	decoded := events[0]
	require.Equal(t, original.Event, decoded.Event)
	require.Equal(t, original.ID, decoded.ID)
	require.Equal(t, original.Retry, decoded.Retry)

	data, err := io.ReadAll(decoded.Data)
	require.NoError(t, err)
	require.Equal(t, "test data", string(data))
}

func TestSSEventEncodeDecode_MultipleEventsRoundTrip(t *testing.T) {
	events := []SSEvent{
		{
			Event: "event1",
			Data:  strings.NewReader("data1"),
			ID:    "1",
		},
		{
			Event: "event2",
			Data:  strings.NewReader("data2"),
			ID:    "2",
		},
		{
			Event: "event3",
			Data:  strings.NewReader("data3"),
			ID:    "3",
		},
	}

	var buf bytes.Buffer
	for _, event := range events {
		err := SSEventEncode(&buf, event)
		require.NoError(t, err)
	}

	decoded, err := SSEventDecode(&buf)
	require.NoError(t, err)
	require.Len(t, decoded, 3)

	for i, event := range decoded {
		require.Equal(t, events[i].Event, event.Event)
		require.Equal(t, events[i].ID, event.ID)
	}
}

func TestSSEventDecode_BOMHandling(t *testing.T) {
	// UTF-8 BOM followed by event data
	input := "\xEF\xBB\xBFdata: test with BOM\n\n"
	events, err := SSEventDecode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)

	data, err := io.ReadAll(events[0].Data)
	require.NoError(t, err)
	require.Equal(t, "test with BOM", string(data))
}

func TestSSEventDecode_CRLFLineEndings(t *testing.T) {
	// Test with CRLF line endings
	input := "event: test\r\ndata: line1\r\ndata: line2\r\n\r\n"
	events, err := SSEventDecode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)

	require.Equal(t, "test", events[0].Event)
	data, err := io.ReadAll(events[0].Data)
	require.NoError(t, err)
	require.Equal(t, "line1\nline2", string(data))
}

func TestSSEventDecode_CRLineEndings(t *testing.T) {
	// Test with CR-only line endings
	input := "event: test\rdata: single CR\r\r"
	events, err := SSEventDecode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)

	require.Equal(t, "test", events[0].Event)
	data, err := io.ReadAll(events[0].Data)
	require.NoError(t, err)
	require.Equal(t, "single CR", string(data))
}

func TestSSEventDecode_IDWithNull(t *testing.T) {
	// ID field containing NULL should be ignored
	input := "id: test\x00null\ndata: test\n\n"
	events, err := SSEventDecode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	// ID should be empty because it contained NULL
	require.Equal(t, "", events[0].ID)
}

func TestSSEventDecode_IDWithoutNull(t *testing.T) {
	// Normal ID should work
	input := "id: valid-id-123\ndata: test\n\n"
	events, err := SSEventDecode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "valid-id-123", events[0].ID)
}

func TestSSEventDecode_NegativeRetry(t *testing.T) {
	// Negative retry values should be ignored
	input := "retry: -100\ndata: test\n\n"
	events, err := SSEventDecode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, uint(0), events[0].Retry)
}

func TestSSEventDecode_RetryWithNonDigits(t *testing.T) {
	// Retry field with non-digit characters should be ignored
	input := "retry: 100ms\ndata: test\n\n"
	events, err := SSEventDecode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, uint(0), events[0].Retry)
}

func TestSSEventDecode_MixedLineEndings(t *testing.T) {
	// Test with mixed line endings
	input := "event: test\r\ndata: line1\ndata: line2\rdata: line3\n\n"
	events, err := SSEventDecode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)

	data, err := io.ReadAll(events[0].Data)
	require.NoError(t, err)
	require.Equal(t, "line1\nline2\nline3", string(data))
}

func TestSSEventDecode_DataFieldWithLeadingSpace(t *testing.T) {
	// Per SSE spec, first space after colon should be removed
	input := "data: with space\ndata:without space\n\n"
	events, err := SSEventDecode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)

	data, err := io.ReadAll(events[0].Data)
	require.NoError(t, err)
	require.Equal(t, "with space\nwithout space", string(data))
}

func TestSSEventDecode_DataFieldWithMultipleLeadingSpaces(t *testing.T) {
	// Only first space should be removed
	input := "data:  two spaces\n\n"
	events, err := SSEventDecode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)

	data, err := io.ReadAll(events[0].Data)
	require.NoError(t, err)
	require.Equal(t, " two spaces", string(data))
}

func TestSSEventDecode_EmptyDataField(t *testing.T) {
	// Empty data field
	input := "data:\n\n"
	events, err := SSEventDecode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)

	data, err := io.ReadAll(events[0].Data)
	require.NoError(t, err)
	require.Equal(t, "", string(data))
}
