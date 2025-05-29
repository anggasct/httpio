// Package client implements the internal HTTP request/response handling
package client

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strconv"
	"strings"
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
func Stream(r *Response, handler func([]byte) error, opts ...StreamOption) error {
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
func StreamLines(r *Response, handler func([]byte) error, opts ...StreamOption) error {
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
	}

	for scanner.Scan() {
		line := scanner.Bytes()
		if handlerErr := handler(line); handlerErr != nil {
			return handlerErr
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

// StreamJSON processes a response stream as JSON objects with the provided handler function.
// The handler is called for each JSON object. If the handler returns an error,
// streaming stops and the error is returned.
func StreamJSON(r *Response, handler func(json.RawMessage) error, opts ...StreamOption) error {
	if r.Body == nil {
		return errors.New("response body is nil")
	}
	defer r.Body.Close()

	options := defaultStreamOptions()
	for _, opt := range opts {
		opt(options)
	}

	decoder := json.NewDecoder(r.Body)
	for {
		var raw json.RawMessage
		err := decoder.Decode(&raw)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if handlerErr := handler(raw); handlerErr != nil {
			return handlerErr
		}
	}
}

// StreamInto processes a response stream as JSON objects and unmarshals each object into a new
// instance of the provided type, then passes it to the handler function.
func StreamInto(r *Response, handler interface{}, opts ...StreamOption) error {
	if r.Body == nil {
		return errors.New("response body is nil")
	}
	defer r.Body.Close()

	options := defaultStreamOptions()
	for _, opt := range opts {
		opt(options)
	}

	handlerValue := reflect.ValueOf(handler)
	if handlerValue.Kind() != reflect.Func {
		return errors.New("handler must be a function")
	}

	handlerType := handlerValue.Type()
	if handlerType.NumIn() != 1 || handlerType.NumOut() != 1 || handlerType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
		return errors.New("handler must have the signature func(T) error")
	}

	elemType := handlerType.In(0)
	isPtr := elemType.Kind() == reflect.Ptr

	decoder := json.NewDecoder(r.Body)
	for {
		var elem reflect.Value
		if isPtr {
			elem = reflect.New(elemType.Elem())
		} else {
			elem = reflect.New(elemType).Elem()
		}

		err := decoder.Decode(elem.Interface())
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		results := handlerValue.Call([]reflect.Value{elem})
		errInterface := results[0].Interface()
		if errInterface != nil {
			return errInterface.(error)
		}
	}
}

// Event represents a Server-Sent Event
type Event struct {
	ID    string
	Event string
	Data  string
	Retry int
}

// EventSourceHandler handles incoming Server-Sent Events
type EventSourceHandler interface {
	OnEvent(event Event) error
}

// EventSourceFullHandler extends EventSourceHandler with lifecycle methods
type EventSourceFullHandler interface {
	EventSourceHandler
	OnOpen() error
	OnClose() error
}

// EventHandlerFunc is a function type for handling SSE events
type EventHandlerFunc func(event Event) error

// OnEvent implements EventSourceHandler interface
func (f EventHandlerFunc) OnEvent(event Event) error {
	return f(event)
}

// EventFullHandlerFunc represents a function-based handler with lifecycle support
type EventFullHandlerFunc struct {
	OnEventFunc func(event Event) error
	OnOpenFunc  func() error
	OnCloseFunc func() error
}

// OnEvent implements EventSourceHandler interface
func (l *EventFullHandlerFunc) OnEvent(event Event) error {
	if l.OnEventFunc != nil {
		return l.OnEventFunc(event)
	}
	return nil
}

// OnOpen implements EventSourceFullHandler interface
func (l *EventFullHandlerFunc) OnOpen() error {
	if l.OnOpenFunc != nil {
		return l.OnOpenFunc()
	}
	return nil
}

// OnClose implements EventSourceFullHandler interface
func (l *EventFullHandlerFunc) OnClose() error {
	if l.OnCloseFunc != nil {
		return l.OnCloseFunc()
	}
	return nil
}

// StreamSSE processes a Server-Sent Events stream with the provided handler.
func StreamSSE(reader io.ReadCloser, handler EventSourceHandler) error {
	defer reader.Close()

	if lifecycleHandler, ok := handler.(EventSourceFullHandler); ok {
		if handlerErr := lifecycleHandler.OnOpen(); handlerErr != nil {
			return handlerErr
		}
		defer lifecycleHandler.OnClose()
	}

	scanner := bufio.NewScanner(reader)
	var event Event
	var data strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			if data.Len() > 0 {
				event.Data = data.String()
				if handlerErr := handler.OnEvent(event); handlerErr != nil {
					return handlerErr
				}
				event = Event{}
				data.Reset()
			}
			continue
		}

		if strings.HasPrefix(line, ":") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		field := parts[0]
		var value string
		if len(parts) > 1 {
			value = strings.TrimPrefix(parts[1], " ")
		}

		switch field {
		case "event":
			event.Event = value
		case "id":
			event.ID = value
		case "retry":
			retry, err := strconv.Atoi(value)
			if err == nil {
				event.Retry = retry
			}
		case "data":
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(value)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
