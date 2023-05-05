// Package sabot implements context logging with json output
package sabot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/pkg/errors"
)

// Fields are key value pairs
type Fields map[string]any

// LogKey is a unique to this package key for use with context Value
type LogKey struct{}

// Sabot is a structured logger
type Sabot struct {
	Writer    io.Writer
	AltWriter io.Writer
}

// Info logs info level events
func (sabot *Sabot) Info(ctx context.Context, msg string, kv ...any) {

	sabot.log(ctx, "info", msg, kv)
}

// Error logs error level events
func (sabot *Sabot) Error(ctx context.Context, msg string, err error, kv ...any) {

	kv = append(kv, "error", fmt.Sprintf("%+v", err))
	sabot.log(ctx, "error", msg, kv)
}

// WithFields adds log fields to a given context
func WithFields(ctx context.Context, kv ...any) context.Context {

	ctxFields := copyFields(ctx)
	kvFields := newFields(kv)

	for key, val := range kvFields {
		ctxFields[key] = val
	}

	return context.WithValue(ctx, LogKey{}, ctxFields)
}

// GetFields gets log fields from a given context
func GetFields(ctx context.Context) Fields {

	val := ctx.Value(LogKey{})
	if val == nil {
		return Fields{}
	}

	fields, ok := val.(Fields)
	if !ok {
		fields = Fields{
			logErrorKey: fmt.Sprintf("failed to assert type Fields on %#v", val),
		}
	}
	return fields
}

//
// unexported
//

func (sabot *Sabot) log(ctx context.Context, level, msg string, kv []any) {

	// Todo: redaction
	// Todo: truncation

	ctxFields := GetFields(ctx)
	kvFields := newFields(kv)

	// silently overwrite kv from ctx and boilerplate when duplicate key

	for key, val := range ctxFields {
		kvFields[key] = val
	}

	kvFields["msg"] = msg
	kvFields["level"] = level
	kvFields["ts"] = time.Now()

	// marshal and try to emit something in case of trouble

	data, err := json.Marshal(kvFields)
	if err != nil {
		// hard to trigger since newFields returns valid
		err = errors.Wrapf(err, "failed to marshal log message")
		data = []byte(fmt.Sprintf(`{"%s": "%+v", "msg": "%#v"}`, logErrorKey, err, kvFields))
	}

	_, err = sabot.Writer.Write(append(data, []byte("\n")...))
	if err != nil && sabot.AltWriter != nil {
		err = errors.Wrapf(err, "failed to write")
		_, _ = fmt.Fprintf(sabot.AltWriter, "%s: %+v with fields %#v\n", logErrorKey, err, kvFields)
	}
}

const (
	logErrorKey string = "logerror"
)

func logErrorFields(err error, kv []any) Fields {

	return Fields{
		logErrorKey: fmt.Sprintf("%+v", err),
		"keyvals":   fmt.Sprintf("%#v", kv),
	}
}

func newFields(kv []any) Fields {

	if len(kv)%2 != 0 {
		err := errors.Errorf("cannot create fields from odd count")
		return logErrorFields(err, kv)
	}

	fields := Fields{}
	for i := 0; i < len(kv); i += 2 {

		key, ok := kv[i].(string)
		if !ok {
			err := errors.Errorf("non-string field key: %#v", kv[i])
			return logErrorFields(err, kv)
		}

		var err error
		fields[key], err = marshalUnknown(kv[i+1])
		if err != nil {
			delete(fields, key)
			for ek, ev := range logErrorFields(err, kv) {
				fields[ek] = ev
			}
		}
	}

	return fields
}

func marshalUnknown(obj any) (any, error) {

	switch obj.(type) {
	case string, []byte, int, int64, float64, time.Time, time.Duration:
		return obj, nil
	default:
		data, err := json.Marshal(obj)
		if err != nil {
			err = errors.Wrapf(err, "failed to marshal: %#v", obj)
			return logErrorKey, err
		}
		return string(data), nil
	}
}

func copyFields(ctx context.Context) Fields {

	cp := Fields{}
	for key, value := range GetFields(ctx) {
		cp[key] = value
	}

	return cp
}
