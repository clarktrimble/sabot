// Package main shows an example of logging with sabot.
package main

import (
	"context"
	"os"

	"github.com/pkg/errors"

	"github.com/clarktrimble/sabot"
)

var (
	version string
)

type Config struct {
	Version string        `json:"version"`
	Logger  *sabot.Config `json:"logger"`
}

func main() {

	// usually load config via launch(envconfig), but literal for demo

	cfg := &Config{
		Version: version,
		Logger: &sabot.Config{
			MaxLen: 99,
		},
	}

	// create logger

	lgr := cfg.Logger.New(os.Stderr)

	// usually set run id with hondo(rand), but literal for demo

	ctx := lgr.WithFields(context.Background(), "run_id", "123123123")

	// log stuff, yay

	lgr.Info(ctx, "logloglog starting", "config", cfg)
	lgr.Error(ctx, "failed to, you know ..", errors.Errorf("oops"))
}

// output:

/*
$ bin/logloglog 2>&1 | jq --slurp
[
  {
    "config": "{\"version\":\"config.11.8a5e577\",\"logger\":{\"max_len\":99}}",
    "level": "info",
    "msg": "logloglog starting",
    "run_id": "123123123",
    "ts": "2023-11-25T21:20:54.758434441Z"
  },
  {
    "error": "oops\nmain.main\n\t/home/trimble/proj/sabot/examples/logloglog/main.go:38\nruntime.main\n\t/--truncated--",
    "level": "error",
    "msg": "failed to, you know ..",
    "run_id": "123123123",
    "ts": "2023-11-25T21:20:54.758722223Z"
  }
]
*/
