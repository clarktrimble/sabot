
# Sabot

Structured Contextual Logging for Golang

![sabot-sketch-small](https://user-images.githubusercontent.com/5055161/236526017-ab7fa549-2230-4088-a22e-aee58f586af7.png)

## Why?

Why yet another Golang module for logging?  Svelte invocation for a start:

```go
  lgr := cfg.Logger.New(os.Stderr)
  ctx := lgr.WithFields(context.Background(), "run_id", "123123123")

  lgr.Info(ctx, "logloglog starting", "config", cfg)
  lgr.Error(ctx, "failed to, you know ..", errors.Errorf("oops"))
```

(from examples/logloglog)

After positional, any additional parameters are interpreted as key-value pairs to be included in the message.
The approach works well in practice and logging statements can contribute to more readable code!

Let's have a look at the output:

```bash
$ bin/logloglog 2>&1 | jq --slurp
```

```json
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
```

Additional features elaborated below! :)

[slog](https://go.dev/blog/slog) is now a thing in the standard library!
Cool to see the same signature for lgr.Info here as [lgr.InfoContext](https://pkg.go.dev/log/slog#Logger.InfoContext).
After a quick read-thru, I'm left with the impression that sabot and slog, while sharing an approach to structured input, are differently focused.

`sabot` aims to implement a small interface, simply.
`slog`, as part of the standard library, has a lot more water to carry and views to consider.
Notably, sabot accumulates key-values via context, where slog wants to duplicate the logger object.
(Contextual logging looks to be left as an exercise for a handler in slog.)
I am curious to try the slog's json handler in sabot.


## Virtues of Contextual Logging

It's nice to use context for .. ahh, the context of the log message.  For example, if:

    ctx = logger.WithFields(ctx, "request_id", rg.String(7))


is added to a request's context in a middleware, `request_id` can be included in any subsequent logs relating to it.
This can be very handy indeed when the time comes to pivot in Kibana, etc.

Accumulating log fields via context is more flexible than in a duplicated logger object.
Say for example, you want to log something about requests from an api's service layer.
All this happens long after the service layer is instantiated with it's logger.

For example: (from [respond.go](https://github.com/clarktrimble/delish/blob/main/respond.go#L26))

```go
func (rp *Respond) NotOk(ctx context.Context, code int, err error) {

  header(rp.Writer, code)

  rp.Logger.Error(ctx, "returning error to client", err)
  rp.WriteObjects(ctx, map[string]any{"error": fmt.Sprintf("%v", err)})
}
```

In the helper shown, `Logger` comes to us from a service-layer, created on startup.
`ctx`, which is created much later with the request, carries `request_id` thanks to a middleware.
A-and the request itself will have been logged with the same id, allowing for correlation :)

When I first began using a contextual approach, I _was_ a little troubled by the need to pass in context to every function that logs.
In practice it's never been a problem and usually I find handling an error and other loggable situations can be kept toward the top of the stack.

## Structured Output

Json is implemented here and I'm interested in adding a lightweight approach OpenTelemetry.

## Best Effort

Sabot will do it's best to emit something, but the priority is to stay out of the way and, where unavoidable, fail gracefully.

### Truncation

Logging request bodies as seen above can be very helpful, especially when troubleshooting a system that's just coming together.
Simply truncating when things get out of hand strikes a nice balance:

    "foo":   `["bar","bar","bar","bar","bar",--truncated--`,

## Unoptimized

Yet!

Really though, sabot is not likely to turn into a high performance logging solution.
It's focused on readability in the code and producing logs for efficient troubleshooting of system issues. 

## Transport Free

Sabot takes an `io.Writer`.
I usually go with `os.Stderr` with the idea that container infrastructure will take it from there.

Where transport is needed to get logs over to ingestion, I've had very good luck with [github.com/fatih/pool](https://github.com/fatih/pool).

## Small Public Interface

Occasionally, logging from a module _is_ what you need.  The middleware examples above, for instance.

A reasonable interface might look like:

    type logger interface {
      Info(ctx context.Context, msg string, kv ...any)
      Error(ctx context.Context, msg string, err error, kv ...any)
      WithFields(ctx context.Context, kv ...any) context.Context
    }

Opinions vary widely, and:

<https://opentelemetry.io/docs/reference/specification/logs/data-model/#displaying-severity>

<https://dave.cheney.net/2015/11/05/lets-talk-about-logging>

Offer some interesting perspectives.  A pull request for Debug will be most welcome!

## The Art of Logging

Logging what you need when you're troubleshooting may take a few passes.
Not too cold and not too hot.

I like an iterative approach:
 - log some stuff as reasonable in dev
 - add, remove, and edit logging statements in lab
 - fine tune as necessary in prod

Devops ftw! :)

## Golang (Anti) Idioms

I dig the Golang community, but I might be a touch rouge with:

  - multi-char variable names
  - named return parameters (unless awkward)
  - only cap first letter of acronyms
  - liberal use of vertical space
  - BDD/DSL testing

All in the name of readability, which of course, tends towards the subjective.

## Concurrent Use

I'm under the impression that `copyFields` in `withFields` makes Sabot safe for concurrent use.
I have used a similar approach to reliably log the noteworthy events of many http requests.

## License

This is free and unencumbered software released into the public domain.

Anyone is free to copy, modify, publish, use, compile, sell, or
distribute this software, either in source code form or as a compiled
binary, for any purpose, commercial or non-commercial, and by any
means.

In jurisdictions that recognize copyright laws, the author or authors
of this software dedicate any and all copyright interest in the
software to the public domain. We make this dedication for the benefit
of the public at large and to the detriment of our heirs and
successors. We intend this dedication to be an overt act of
relinquishment in perpetuity of all present and future rights to this
software under copyright law.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS BE LIABLE FOR ANY CLAIM, DAMAGES OR
OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
OTHER DEALINGS IN THE SOFTWARE.

For more information, please refer to <http://unlicense.org/>
