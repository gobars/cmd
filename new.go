package cmd

import (
	"sync"
)

// NewCmd creates a new Cmd for the given command name and arguments. The command
// is not started until Start is called. Output buffering is on, streaming output
// is off. To control output, use NewCmdOptions instead.
func NewCmd(cmdparts ...string) *Cmd {
	var args []string

	if len(cmdparts) == 1 {
		args = []string{}
	} else {
		args = cmdparts[1:]
	}

	name := cmdparts[0]

	return &Cmd{
		Name:     name,
		Args:     args,
		buffered: true,
		Mutex:    &sync.Mutex{},
		status: Status{
			Cmd:      name,
			PID:      0,
			Complete: false,
			Exit:     -1,
			Error:    nil,
			Runtime:  0,
		},
		doneChan: make(chan struct{}),
	}
}

// NewCmdOptions creates a new Cmd with options. The command is not started
// until Start is called.
func NewCmdOptions(options Options, cmdparts ...string) *Cmd {
	c := NewCmd(cmdparts...)
	c.applyOption(options)

	return c
}

func (c *Cmd) applyOption(options Options) {
	c.buffered = options.Buffered
	if options.Streaming {
		c.Stdout = make(chan string, DefaultStreamChanSize)
		c.Stderr = make(chan string, DefaultStreamChanSize)
	}

	c.timeout = options.Timeout

	if options.StdinEnabled {
		c.Stdin = make(chan string)
	}

	c.Env = options.Env
}
