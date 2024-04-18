package sabot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSabot(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sabot Suite")
}

var _ = Describe("Sabot", func() {

	var (
		ctx context.Context
		lgr *Sabot
	)

	Describe("creating a logger from config", func() {
		var (
			cfg *Config
		)

		JustBeforeEach(func() {
			lgr = cfg.New(os.Stderr)
		})

		When("all is well", func() {
			BeforeEach(func() {
				cfg = &Config{
					MaxLen: 99,
				}
			})

			It("should setup the logger", func() {
				Expect(lgr.MaxLen).To(Equal(99))
				Expect(lgr.Writer).To(Equal(os.Stderr))
			})
		})
	})

	Describe("getting and storing fields", func() {
		var (
			fields Fields
		)

		BeforeEach(func() {
			ctx = context.Background()
			lgr = &Sabot{
				MaxLen: 0,
			}
		})

		Context("in a ctx", func() {

			JustBeforeEach(func() {
				fields = lgr.GetFields(ctx)
			})

			When("nothing in ctx", func() {
				It("should return an empty slice", func() {
					Expect(fields).To(Equal(Fields{}))
				})
			})

			When("something stored in ctx", func() {
				BeforeEach(func() {
					ctx = lgr.WithFields(ctx, "foo", "bar")
				})

				It("should return something", func() {
					Expect(fields).To(Equal(Fields{"foo": "bar"}))
				})

				When("another thing is added to the same ctx", func() {
					BeforeEach(func() {
						ctx = lgr.WithFields(ctx, "another", "thing")
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
					ctx = lgr.WithFields(ctx, "foo", "bar", "odd")
				})

				It("should return logerror", func() {
					replace(fields)
					Expect(fields).To(Equal(Fields{
						"logerror": "cannot create fields from odd count",
						"keyvals":  "keyvals replaced for test",
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
				msg string
				err error
				kv  []any
			)

			BeforeEach(func() {
				buf = &bytes.Buffer{}
				lgr = &Sabot{
					Writer: buf,
					MaxLen: 0,
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

			Context("at debug level", func() {

				JustBeforeEach(func() {
					lgr.Debug(ctx, msg, kv...)
				})

				When("debug is enabled", func() {
					BeforeEach(func() {
						lgr.EnableDebug = true
					})
					It("should write the message, level, and ts", func() {
						Expect(delog(buf)).To(Equal(Fields{
							"level": "debug",
							"msg":   "a noteworthy occurrence",
							"ts":    "nowish",
						}))
					})
				})

				When("debug is not enabled", func() {
					It("should skip", func() {
						Expect(delog(buf)).To(BeEmpty())
					})
				})
			})

			Context("at trace level", func() {

				JustBeforeEach(func() {
					lgr.Trace(ctx, msg, kv...)
				})

				When("trace is enabled", func() {
					BeforeEach(func() {
						lgr.EnableTrace = true
					})
					It("should write the message, level, and ts", func() {
						Expect(delog(buf)).To(Equal(Fields{
							"level": "trace",
							"msg":   "a noteworthy occurrence",
							"ts":    "nowish",
						}))
					})
				})

				When("trace is not enabled", func() {
					It("should skip", func() {
						Expect(delog(buf)).To(BeEmpty())
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
						ctx = lgr.WithFields(ctx, "app_id", "testo", "app_grp", "global")
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

				When("no ctx fields and object val in kv larger than max", func() {
					BeforeEach(func() {
						kv = []any{"foo", []string{"bar", "bar", "bar", "bar", "bar", "baaaaaarrrrr"}}
						lgr.MaxLen = 44
					})

					It("should write the message, level, ts, and truncated object", func() {
						lgd := delog(buf)

						Expect(lgd["foo"]).To(HaveLen(44))
						Expect(lgd).To(Equal(Fields{
							"level": "info",
							"msg":   "a noteworthy occurrence",
							"ts":    "nowish",
							"foo":   `["bar","bar","bar","bar","bar",--truncated--`,
						}))
					})
				})

				When("ctx fields and kv fields", func() {
					BeforeEach(func() {
						ctx = lgr.WithFields(ctx, "app_id", "testo")
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
						ctx = lgr.WithFields(ctx, "app_id", "testo", "level", "warn21")
						kv = []any{"foo", "bar", "app_id", "producto", "level", "warn22"} //nolint: misspell
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
							"keyvals":  "keyvals replaced for test",
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
							"keyvals":  "keyvals replaced for test",
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
							"keyvals":  "keyvals replaced for test",
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

	if buf.Len() == 0 {
		return
	}

	// marshal logged data to map

	logged = Fields{}
	err := json.Unmarshal(buf.Bytes(), &logged)
	Expect(err).ToNot(HaveOccurred())

	// check time here and rewrite for testability

	loggedAt, err := time.Parse(time.RFC3339, logged["ts"].(string))
	Expect(err).ToNot(HaveOccurred())
	Expect(loggedAt).To(BeTemporally("~", time.Now(), 9*time.Millisecond))
	logged["ts"] = "nowish"

	replace(logged)
	return
}

func replace(logged Fields) {

	logerror, ok := logged["logerror"]
	if ok {
		logged["logerror"] = strings.Split(logerror.(string), "\n")[0] //nolint: forcetypeassert
	}

	_, ok = logged["keyvals"]
	if ok {
		logged["keyvals"] = "keyvals replaced for test"
	}
}

type failWriter struct{}

func (fw failWriter) Write(p []byte) (n int, err error) {

	err = fmt.Errorf("oops")
	return
}
