package sabot_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/clarktrimble/sabot"
)

func TestSabot(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sabot Suite")
}

var _ = Describe("Sabot", func() {

	Describe("getting and storing fields", func() {
		var (
			ctx    context.Context
			fields Fields
		)

		BeforeEach(func() {
			ctx = context.Background()
		})

		Context("in a ctx", func() {

			JustBeforeEach(func() {
				fields = GetFields(ctx)
			})

			When("nothing in ctx", func() {
				It("should return an empty slice", func() {
					Expect(fields).To(Equal(Fields{}))
				})
			})

			When("something stored in ctx", func() {
				BeforeEach(func() {
					ctx = WithFields(ctx, "foo", "bar")
				})

				It("should return something", func() {
					Expect(fields).To(Equal(Fields{"foo": "bar"}))
				})

				When("another thing is added to the same ctx", func() {
					BeforeEach(func() {
						ctx = WithFields(ctx, "another", "thing")
					})

					It("should return both", func() {
						Expect(fields).To(Equal(Fields{
							"foo":     "bar",
							"another": "thing",
						}))
					})
				})
			})

			When("odd count stored in ctx", func() {
				BeforeEach(func() {
					ctx = WithFields(ctx, "foo", "bar", "odd")
				})

				It("should return logerror", func() {
					redact(&fields)
					Expect(fields).To(Equal(Fields{
						"logerror": "cannot create fields from odd count",
						"keyvals":  "keyvals redacted for test",
					}))
				})
			})

			When("non-Fields stored in ctx", func() {
				BeforeEach(func() {
					ctx = context.WithValue(ctx, LogKey{}, "garbage")
				})

				It("should return logerror", func() {
					Expect(fields).To(Equal(Fields{"logerror": `failed to assert type Fields on "garbage"`}))
				})
			})
		})

		Describe("logging an event", func() {
			var (
				buf *bytes.Buffer
				ctx context.Context
				msg string
				err error
				kv  []any
				lgr Sabot
			)

			BeforeEach(func() {
				buf = &bytes.Buffer{}
				lgr = Sabot{
					Writer: buf,
				}
				ctx = context.Background()
				msg = "a noteworthy occurrence"
				err = nil
				kv = nil
			})

			Context("at error level", func() {

				JustBeforeEach(func() {
					lgr.Error(ctx, msg, err, kv...)
				})

				When("no ctx fields and no kv fields", func() {
					BeforeEach(func() {
						err = fmt.Errorf("oops")
					})
					It("should write the message, level, ts, and error", func() {
						Expect(delog(buf)).To(Equal(Fields{
							"level": "error",
							"msg":   "a noteworthy occurrence",
							"ts":    "nowish",
							"error": "oops",
						}))
					})
				})
			})

			Context("at info level", func() {

				JustBeforeEach(func() {
					lgr.Info(ctx, msg, kv...)
				})

				When("no ctx fields and no kv fields", func() {
					It("should write the message, level, and ts", func() {
						Expect(delog(buf)).To(Equal(Fields{
							"level": "info",
							"msg":   "a noteworthy occurrence",
							"ts":    "nowish",
						}))
					})
				})

				When("ctx fields and no kv fields", func() {
					BeforeEach(func() {
						ctx = WithFields(ctx, "app_id", "testo", "app_grp", "global")
					})

					It("should write the message, level, ts, and fields", func() {
						Expect(delog(buf)).To(Equal(Fields{
							"level":   "info",
							"msg":     "a noteworthy occurrence",
							"ts":      "nowish",
							"app_id":  "testo",
							"app_grp": "global",
						}))
					})
				})

				When("no ctx fields and kv fields", func() {
					BeforeEach(func() {
						kv = []any{"foo", "bar", "cid", 777}
					})

					It("should write the message, level, ts, and fields", func() {
						Expect(delog(buf)).To(Equal(Fields{
							"level": "info",
							"msg":   "a noteworthy occurrence",
							"ts":    "nowish",
							"foo":   "bar",
							"cid":   float64(777),
						}))
					})
				})

				When("no ctx fields and object val in kv", func() {
					BeforeEach(func() {
						kv = []any{"foo", []string{"bar"}}
					})

					It("should write the message, level, ts, and marshalled object", func() {
						Expect(delog(buf)).To(Equal(Fields{
							"level": "info",
							"msg":   "a noteworthy occurrence",
							"ts":    "nowish",
							"foo":   `["bar"]`,
						}))
					})
				})

				When("ctx fields and kv fields", func() {
					BeforeEach(func() {
						ctx = WithFields(ctx, "app_id", "testo")
						kv = []any{"foo", "bar"}
					})

					It("should write the message, level, ts, and fields", func() {
						Expect(delog(buf)).To(Equal(Fields{
							"level":  "info",
							"msg":    "a noteworthy occurrence",
							"ts":     "nowish",
							"app_id": "testo",
							"foo":    "bar",
						}))
					})
				})

				When("ctx fields and kv fields overlap each other and boilerplate", func() {
					BeforeEach(func() {
						ctx = WithFields(ctx, "app_id", "testo", "level", "warn21")
						kv = []any{"foo", "bar", "app_id", "producto", "level", "warn22"}
					})

					It("should write the message, level, ts, and fields", func() {
						Expect(delog(buf)).To(Equal(Fields{
							"level":  "info",
							"msg":    "a noteworthy occurrence",
							"ts":     "nowish",
							"app_id": "testo",
							"foo":    "bar",
						}))
					})
				})

				When("no ctx fields and kv odd fields", func() {
					BeforeEach(func() {
						kv = []any{"foo", "bar", "odd"}
					})

					It("should write the message, level, ts, and fields", func() {
						Expect(delog(buf)).To(Equal(Fields{
							"level":    "info",
							"msg":      "a noteworthy occurrence",
							"ts":       "nowish",
							"keyvals":  "keyvals redacted for test",
							"logerror": "cannot create fields from odd count",
						}))
					})
				})

				When("no ctx fields and kv non-string key", func() {
					BeforeEach(func() {
						kv = []any{88, "bar"}
					})

					It("should write the message, level, ts, and fields", func() {
						Expect(delog(buf)).To(Equal(Fields{
							"level":    "info",
							"msg":      "a noteworthy occurrence",
							"ts":       "nowish",
							"logerror": "non-string field key: 88",
							"keyvals":  "keyvals redacted for test",
						}))
					})
				})

				When("no ctx fields and kv unmarshallable value", func() {
					BeforeEach(func() {
						kv = []any{"foo", make(chan int)}
					})

					It("should write the message, level, ts, and fields", func() {
						Expect(delog(buf)).To(Equal(Fields{
							"level":    "info",
							"msg":      "a noteworthy occurrence",
							"ts":       "nowish",
							"keyvals":  "keyvals redacted for test",
							"logerror": "json: unsupported type: chan int",
						}))
					})
				})

				When("writer returns error and alternate writer defined", func() {
					var altBuf *bytes.Buffer

					BeforeEach(func() {
						lgr.Writer = failWriter{}

						altBuf = &bytes.Buffer{}
						lgr.AltWriter = altBuf
					})

					It("should write the message, level, ts, and fields", func() {
						Expect(altBuf.String()).To(HavePrefix("logerror"))
					})
				})
			})

		})
	})
})

func delog(buf *bytes.Buffer) (logged Fields) {

	// marshal logged data to map

	logged = Fields{}
	err := json.Unmarshal(buf.Bytes(), &logged)
	Expect(err).To(BeNil())

	// check time here and rewrite for testability

	loggedAt, err := time.Parse(time.RFC3339, logged["ts"].(string))
	Expect(err).To(BeNil())
	Expect(loggedAt).To(BeTemporally("~", time.Now(), 9*time.Millisecond))
	logged["ts"] = "nowish"

	redact(&logged)
	return
}

func redact(asdf *Fields) {

	logged := *asdf
	logerror, ok := logged["logerror"]
	if ok {
		logged["logerror"] = strings.Split(logerror.(string), "\n")[0]
	}
	logerror, ok = logged["keyvals"]
	if ok {
		logged["keyvals"] = "keyvals redacted for test"
	}
}

type failWriter struct{}

func (fw failWriter) Write(p []byte) (n int, err error) {

	err = fmt.Errorf("oops")
	return
}
