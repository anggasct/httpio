package stream

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/anggasct/httpio/internal/client"
)

// StreamOption represents options for stream processing
type StreamOption func(*streamOptions)

type streamOptions struct {
	buffSize      int
	contentType   string
	delimiterStr  string
	delimiterByte byte
}

// WithBufferSize sets the buffer size for stream reading
func WithBufferSize(size int) StreamOption {
	return func(o *streamOptions) {
		o.buffSize = size
	}
}

// WithDelimiter sets the delimiter for line-based stream reading
func WithDelimiter(delimiter string) StreamOption {
	return func(o *streamOptions) {
		o.delimiterStr = delimiter
	}
}

// WithByteDelimiter sets a byte delimiter for stream reading
func WithByteDelimiter(delimiter byte) StreamOption {
	return func(o *streamOptions) {
		o.delimiterByte = delimiter
		o.delimiterStr = string(delimiter)
	}
}

// WithContentType sets the expected content type for the stream
func WithContentType(contentType string) StreamOption {
	return func(o *streamOptions) {
		o.contentType = contentType
	}
}

// defaultStreamOptions returns the default stream options
func defaultStreamOptions() *streamOptions {
	return &streamOptions{
		buffSize:      4096,
		contentType:   "",
		delimiterStr:  "\n",
		delimiterByte: '\n',
	}
}

// Stream processes a response stream with the provided handler function.
// The handler is called for each chunk of data. If the handler returns an error,
// streaming stops and the error is returned.
func Stream(r *client.Response, handler func([]byte) error, opts ...StreamOption) error {
	if r.Body == nil {
		return errors.New("response body is nil")
	}
	defer r.Body.Close()

	options := defaultStreamOptions()
	for _, opt := range opts {
		opt(options)
	}

	if options.contentType != "" {
		contentType := r.Header.Get("Content-Type")
		if !strings.Contains(contentType, options.contentType) {
			return errors.New("unexpected content type: " + contentType)
		}
	}

	buffer := make([]byte, options.buffSize)

	for {
		n, err := r.Body.Read(buffer)

		if n > 0 {
			if handlerErr := handler(buffer[:n]); handlerErr != nil {
				return handlerErr
			}
		}

		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

// StreamLines processes a response stream line by line with the provided handler function.
// The handler is called for each line. If the handler returns an error,
// streaming stops and the error is returned.
func StreamLines(r *client.Response, handler func([]byte) error, opts ...StreamOption) error {
	if r.Body == nil {
		return errors.New("response body is nil")
	}
	defer r.Body.Close()

	options := defaultStreamOptions()
	for _, opt := range opts {
		opt(options)
	}

	if options.contentType != "" {
		contentType := r.Header.Get("Content-Type")
		if !strings.Contains(contentType, options.contentType) {
			return errors.New("unexpected content type: " + contentType)
		}
	}
	scanner := bufio.NewScanner(r.Body)

	if options.delimiterStr != "\n" {
		scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
			if atEOF && len(data) == 0 {
				return 0, nil, nil
			}

			if i := strings.Index(string(data), options.delimiterStr); i >= 0 {
				return i + len(options.delimiterStr), data[0:i], nil
			}

			if atEOF {
				return len(data), data, nil
			}

			return 0, nil, nil
		})
	} else if options.delimiterByte != '\n' {
		scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
			for i := 0; i < len(data); i++ {
				if data[i] == options.delimiterByte {
					return i + 1, data[:i], nil
				}
			}

			if atEOF {
				return len(data), data, nil
			}

			return 0, nil, nil
		})
	}

	if options.buffSize > 0 {
		scanner.Buffer(make([]byte, 0, options.buffSize), options.buffSize)
	}
	for scanner.Scan() {
		if err := handler(scanner.Bytes()); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// StreamJSON processes a response stream as JSON objects with the provided handler function.
// The handler is called for each JSON object. If the handler returns an error,
// streaming stops and the error is returned.
// The response should be a newline-delimited JSON stream (NDJSON).
func StreamJSON(r *client.Response, handler func(json.RawMessage) error, opts ...StreamOption) error {
	return StreamLines(r, func(line []byte) error {
		if len(line) == 0 {
			return nil
		}

		var raw json.RawMessage
		if err := json.Unmarshal(line, &raw); err != nil {
			return err
		}

		return handler(raw)
	}, opts...)
}

// StreamInto processes a response stream as JSON objects and unmarshals each object into a new
// instance of the provided type, then passes it to the handler function.
// The handler function should have the signature func(T) error where T is the type
// to unmarshal each JSON object into.
//
// Example usage:
//
//	type User struct {
//	    ID   int    `json:"id"`
//	    Name string `json:"name"`
//	}
//
//	err := resp.StreamInto(func(user User) error {
//	    fmt.Printf("Got user: %s (ID: %d)\n", user.Name, user.ID)
//	    return nil
//	})
func StreamInto(r *client.Response, handler interface{}, opts ...StreamOption) error {
	handlerValue := reflect.ValueOf(handler)
	handlerType := handlerValue.Type()

	if handlerType.Kind() != reflect.Func {
		return errors.New("handler must be a function")
	}

	if handlerType.NumIn() != 1 {
		return errors.New("handler must have exactly one input parameter")
	}

	if handlerType.NumOut() != 1 {
		return errors.New("handler must return exactly one value (error)")
	}

	if !handlerType.Out(0).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return errors.New("handler must return an error")
	}

	paramType := handlerType.In(0)

	return StreamJSON(r, func(raw json.RawMessage) error {
		paramValue := reflect.New(paramType)

		if err := json.Unmarshal(raw, paramValue.Interface()); err != nil {
			return err
		}

		returnValues := handlerValue.Call([]reflect.Value{paramValue.Elem()})
		errorValue := returnValues[0]
		if !errorValue.IsNil() {
			return errorValue.Interface().(error)
		}

		return nil
	}, opts...)
}

// GetStream is a convenience method for making a GET request and streaming the response
func GetStream(c *client.Client, ctx context.Context, path string, handler func([]byte) error, opts ...StreamOption) error {
	resp, err := c.GET(ctx, path)
	if err != nil {
		return err
	}

	return Stream(resp, handler, opts...)
}

// GetStreamLines is a convenience method for making a GET request and streaming the response line by line
func GetStreamLines(c *client.Client, ctx context.Context, path string, handler func([]byte) error, opts ...StreamOption) error {
	resp, err := c.GET(ctx, path)
	if err != nil {
		return err
	}

	return StreamLines(resp, handler, opts...)
}

// GetStreamJSON is a convenience method for making a GET request and streaming the response as JSON
func GetStreamJSON(c *client.Client, ctx context.Context, path string, handler func(json.RawMessage) error, opts ...StreamOption) error {
	resp, err := c.GET(ctx, path)
	if err != nil {
		return err
	}

	return StreamJSON(resp, handler, opts...)
}

// GetStreamInto is a convenience method for making a GET request and streaming the response into a typed handler
func GetStreamInto(c *client.Client, ctx context.Context, path string, handler interface{}, opts ...StreamOption) error {
	resp, err := c.GET(ctx, path)
	if err != nil {
		return err
	}

	return StreamInto(resp, handler, opts...)
}

// SSEEvent represents a Server-Sent Event with all possible fields
type SSEEvent struct {
	Event string
	Data  string
	ID    string
	Retry int
}

// EventSourceHandler is an enhanced handler that receives full SSE event information
type EventSourceHandler func(event SSEEvent) error

// GetSSE is a convenience method for making a GET request to a Server-Sent Events endpoint
// with support for all SSE fields (id, retry, etc.)
func GetSSE(c *client.Client, ctx context.Context, path string, handler EventSourceHandler) error {
	resp, err := c.GET(ctx, path)
	if err != nil {
		return err
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/event-stream") {
		resp.Close()
		return errors.New("unexpected content type: " + contentType)
	}

	return streamSSE(resp.Body, handler)
}

// streamSSE processes a Server-Sent Events stream
// This is a simplified and optimized implementation for handling SSE protocol
func streamSSE(body io.ReadCloser, handler EventSourceHandler) error {
	if body == nil {
		return errors.New("response body is nil")
	}
	defer body.Close()

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 8192), 1048576)

	var (
		event   string
		data    string
		id      string
		retry   int
		hasData bool
	)

	processEvent := func() error {
		if !hasData {
			return nil
		}

		if event == "" {
			event = "message"
		}

		err := handler(SSEEvent{
			Event: event,
			Data:  data,
			ID:    id,
			Retry: retry,
		})

		if err != nil {
			return err
		}

		event = ""
		data = ""
		hasData = false

		return nil
	}

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			if err := processEvent(); err != nil {
				return err
			}
			continue
		}

		if len(line) > 6 && line[0:6] == "event:" {
			event = strings.TrimSpace(line[6:])
		} else if len(line) > 5 && line[0:5] == "data:" {
			value := strings.TrimSpace(line[5:])
			if data != "" {
				data += "\n"
			}
			data += value
			hasData = true
		} else if len(line) > 3 && line[0:3] == "id:" {
			id = strings.TrimSpace(line[3:])
		} else if len(line) > 6 && line[0:6] == "retry:" {
			retryStr := strings.TrimSpace(line[6:])
			if retryStr != "" {
				if parsed, err := strconv.Atoi(retryStr); err == nil && parsed >= 0 {
					retry = parsed
				}
			}
		} else if len(line) > 0 && line[0] == ':' {
			continue
		}
	}

	if err := processEvent(); err != nil {
		return err
	}

	return scanner.Err()
}
