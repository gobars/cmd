package cmd

import (
	"time"
)

// Run runs a cmd.
func Run(cmdparts ...string) (*Cmd, Status) { p := NewCmd(cmdparts...); return p, <-p.Start() }

// BashLiner execute a bash script with line output processing.
func BashLiner(bash string, liner func(line string) bool, optionFns ...OptionFn) (*Cmd, Status) {
	p := NewCmd("bash", "-c", bash)
	option := createOption(optionFns)
	option.Streaming = true
	option.Buffered = false
	p.applyOption(option)
	ch := p.Start()

	for {
		select {
		case val := <-p.Stdout:
			if !liner(val) {
				_ = p.Stop()
			}
		case status := <-ch:
			return p, status
		}
	}
}

// Bash executes a bash scripts.
func Bash(bash string, optionFns ...OptionFn) (*Cmd, Status) {
	p := NewCmd("bash", "-c", bash)
	p.Options(optionFns...)

	return p, <-p.Start()
}

func createOption(optionFns []OptionFn) Options {
	options := Options{Buffered: true}

	for _, fn := range optionFns {
		fn(&options)
	}

	return options
}

// OptionFn alias a function to options function.
type OptionFn func(options *Options)

// SliceAdd add a value to a slice.
func SliceAdd(m []string, v string) []string {
	if m == nil {
		m = make([]string, 0)
	}

	m = append(m, v)

	return m
}

// Env set env to cmd.
func Env(env string) OptionFn { return func(opt *Options) { opt.Env = SliceAdd(opt.Env, env) } }

// Timeout set timeout to cmd.
func Timeout(timeout time.Duration) OptionFn { return func(opt *Options) { opt.Timeout = timeout } }

// Buffered set cmd output should be buffered or not.
func Buffered(buffered bool) OptionFn { return func(opt *Options) { opt.Buffered = buffered } }

// Streaming set cmd output should be streaming or not.
func Streaming() OptionFn { return func(opt *Options) { opt.Streaming = true } }

// Stdin set cmd stdin enabled or not.
func Stdin() OptionFn { return func(opt *Options) { opt.StdinEnabled = true } }

// Options apply some options to cmd.
func (c *Cmd) Options(fns ...OptionFn) { c.applyOption(createOption(fns)) }

// SafeClose close a channel even if it is closed safely without panic.
func SafeClose(ch chan string) (justClosed bool) {
	defer func() {
		if recover() != nil {
			// The return result can be altered
			// in a defer function call.
			justClosed = false
		}
	}()

	// assume ch != nil here.
	close(ch) // panic if ch is closed

	return true // <=> justClosed = true; return
}
