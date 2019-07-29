package cmd

import (
	"time"
)

func Run(cmdparts ...string) (*Cmd, Status) {
	p := NewCmd(cmdparts...)
	return p, <-p.Start()
}

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

type OptionFn func(options *Options)

func Timeout(timeout time.Duration) OptionFn {
	return func(options *Options) { options.Timeout = timeout }
}

func Buffered(buffered bool) OptionFn {
	return func(options *Options) { options.Buffered = buffered }
}

func Streaming() OptionFn {
	return func(options *Options) { options.Streaming = true }
}

func Stdin() OptionFn {
	return func(options *Options) { options.StdinEnabled = true }
}

func (c *Cmd) Options(optionFns ...OptionFn) {
	c.applyOption(createOption(optionFns))
}
func SafeClose(ch chan string) (justClosed bool) {
	defer func() {
		if recover() != nil {
			// The return result can be altered
			// in a defer function call.
			justClosed = false
		}
	}()

	// assume ch != nil here.
	close(ch)   // panic if ch is closed
	return true // <=> justClosed = true; return
}
