package sabot

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// unoptimized

/*
~/proj/sabot$ go test -run=XXX -bench=BenchmarkLog github.com/clarktrimble/sabot
goos: linux
goarch: amd64
pkg: github.com/clarktrimble/sabot
cpu: Intel(R) N95
BenchmarkLog/Info-no_fields-4            1459408               861.2 ns/op          1184 B/op         21 allocs/op
BenchmarkLog/Error-no_fields-4           1000000              1107 ns/op            1349 B/op         26 allocs/op
BenchmarkLog/Info-string_field-4         1000000              1004 ns/op            1312 B/op         23 allocs/op
BenchmarkLog/Error-string_field-4         946570              1258 ns/op            1509 B/op         28 allocs/op
BenchmarkLog/Info-all_the_fields-4        357385              3275 ns/op            4439 B/op         58 allocs/op
BenchmarkLog/Error-all_the_fields-4       326656              3642 ns/op            5227 B/op         63 allocs/op
PASS
ok      github.com/clarktrimble/sabot   7.650s
*/

func BenchmarkLog(b *testing.B) {

	lgr := &Sabot{
		Writer: &nullWriter{},
	}

	ctx := lgr.WithFields(context.Background(), "app_id", "testo", "worker_id", "1234asdf")

	tests := []struct {
		name string
		args []any
	}{
		{
			"no fields",
			nil,
		},
		{
			"string field",
			[]any{
				"string_field", "an important thing",
			},
		},
		{
			"all the fields",
			[]any{
				"string_field", "an important thing",
				"integer_field", 88,
				"float_field", 88.8,
				"bool_field", true,
				"ts_field", time.Time{},
				"duration_field", time.Minute,
				"slice_field", []string{"one", "two"},
				"map_field", map[string]any{"one": 2},
				"obj_field", demo{One: "one", Two: 2},
			},
		},
	}
	for _, tt := range tests {
		b.Run(fmt.Sprintf("Info-%s", tt.name), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					lgr.Info(ctx, "test message", tt.args...)
				}
			})
		})

		b.Run(fmt.Sprintf("Error-%s", tt.name), func(b *testing.B) {
			err := fmt.Errorf("oops")

			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					lgr.Error(ctx, "test message", err, tt.args...)
				}
			})
		})
	}
}

type nullWriter struct{}

func (nw *nullWriter) Write(p []byte) (n int, err error) {
	return
}

type demo struct {
	One string
	Two int
}
