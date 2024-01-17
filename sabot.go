// Package sabot implements contextual logging with json output.
package sabot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	logErrorKey      string = "logerror"
	truncationNotice string = "--truncated--"
)

// Fields are key-value pairs.
type Fields map[string]any

// Config is the configurable fields of Sabot.
type Config struct {
	MaxLen int `json:"max_len" desc:"maximum length that will be logged for any field"`
}

// New creates a Sabot from Config.
func (cfg *Config) New(writer io.Writer) *Sabot {

	return &Sabot{
		MaxLen: cfg.MaxLen,
		Writer: writer,
	}
}

// LogKey is a unique to this package key for use with context Value.
type LogKey struct{}

// Sabot is a structured logger.
type Sabot struct {
	// Writer is where output is written.
	Writer io.Writer
	// AltWriter is where output is written when Writer.Write returns an error.
	AltWriter io.Writer
	// MaxLen is the length at which string field values are truncated.
	MaxLen int
}

// Info logs info level events.
func (sabot *Sabot) Info(ctx context.Context, msg string, kv ...any) {

	sabot.log(ctx, "info", msg, kv)
}

// Error logs error level events.
func (sabot *Sabot) Error(ctx context.Context, msg string, err error, kv ...any) {

	kv = append(kv, "error", fmt.Sprintf("%+v", err))
	sabot.log(ctx, "error", msg, kv)
}

// WithFields adds log fields to a given context.
func (sabot *Sabot) WithFields(ctx context.Context, kv ...any) context.Context {

	return withFields(ctx, kv)
}

// GetFields gets log fields from a given context.
func (sabot *Sabot) GetFields(ctx context.Context) Fields {

	return getFields(ctx)
}

//
// unexported
//

func (sabot *Sabot) log(ctx context.Context, level, msg string, kv []any) {

	now := time.Now().UTC()

	ctxFields := sabot.GetFields(ctx)
	fields := newFields(kv)

	// silently overwrite kv from ctx and boilerplate when duplicate key

	for key, val := range ctxFields {
		fields[key] = val
	}

	fields["msg"] = msg
	fields["level"] = level
	fields["ts"] = now

	fields.truncate(sabot.MaxLen)

	// marshal and try to emit something in case of trouble

	data, err := json.Marshal(fields)
	if err != nil {
		// hard to trigger since newFields returns valid
		err = errors.Wrapf(err, "failed to marshal log message")
		data = []byte(fmt.Sprintf(`{"%s": "%+v", "msg": "%#v"}`, logErrorKey, err, fields))
	}

	_, err = sabot.Writer.Write(append(data, []byte("\n")...))
	if err != nil && sabot.AltWriter != nil {
		err = errors.Wrapf(err, "failed to write")
		_, _ = fmt.Fprintf(sabot.AltWriter, "%s: %+v with fields %#v\n", logErrorKey, err, fields)
	}
}

func withFields(ctx context.Context, kv []any) context.Context {

	fields := copyFields(ctx)
	kvFields := newFields(kv)

	// silently overwrite ctx from kv when duplicate key

	for key, val := range kvFields {
		fields[key] = val
	}

	return context.WithValue(ctx, LogKey{}, fields)
}

func getFields(ctx context.Context) Fields {

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

	// interpret elements of slice as key-value pairs

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
	for key, value := range getFields(ctx) {
		cp[key] = value
	}

	return cp
}

func (fields Fields) truncate(max int) {

	// account for notice length in truncation result

	max -= len(truncationNotice)
	if max < 1 {
		return
	}

	for key, val := range fields {

		str, ok := val.(string)
		if ok && max < len(str) {
			fields[key] = strings.Join([]string{str[:max], truncationNotice}, "")
		}
	}
}
