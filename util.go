package cmd

import (
	"time"
)

func Run(name string, args ...string) (*Cmd, Status) {
	p := NewCmd(name, args...)
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

func Streaming(streaming bool) OptionFn {
	return func(options *Options) { options.Streaming = streaming }
}

func (c *Cmd) Options(optionFns ...OptionFn) {
	c.applyOption(createOption(optionFns))
}