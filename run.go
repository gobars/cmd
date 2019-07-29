package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"syscall"
	"time"
)

func (c *Cmd) run() {
	defer func() {
		c.statusChan <- c.Status() // unblocks Start if caller is waiting
		close(c.doneChan)
	}()

	// /////////////// Setup command
	ctx := context.Background()
	if c.timeout > 0 {
		// Create a new context and add a timeout to it
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel() // The cancel should be deferred so resources are cleaned up
	}
	// Create the command with our context
	cmd := exec.CommandContext(ctx, c.Name, c.Args...)

	// Set process group ID so the cmd and all its children become a new
	// process group. This allows Stop to SIGTERM the cmd's process group
	// without killing this process (i.e. this code here).
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if c.Stdin != nil {
		stdin, _ := cmd.StdinPipe()
		go func() {
			for in := range c.Stdin {
				buf := bytes.NewBufferString(in)
				buf.WriteString("\n")
				_, _ = stdin.Write(buf.Bytes())
			}
			_ = stdin.Close()
		}()
	}

	// Write stdout and stderr to buffers that are safe to read while writing
	// and don't cause a race condition.
	if c.buffered {
		c.stdout = NewOutputBuffer()
		c.stderr = NewOutputBuffer()
	}

	switch {
	case c.buffered && c.Stdout != nil:
		// Buffered and streaming, create both and combine with io.MultiWriter
		cmd.Stdout = io.MultiWriter(NewOutputStream(c.Stdout), c.stdout)
		cmd.Stderr = io.MultiWriter(NewOutputStream(c.Stderr), c.stderr)
	case c.buffered: // Buffered only
		cmd.Stdout = c.stdout
		cmd.Stderr = c.stderr
	case c.Stdout != nil: // Streaming only
		cmd.Stdout = NewOutputStream(c.Stdout)
		cmd.Stderr = NewOutputStream(c.Stderr)
	default: // No output (effectively >/dev/null 2>&1)
		cmd.Stdout = nil
		cmd.Stderr = nil
	}

	// Set the runtime environment for the command as per os/exec.Cmd.
	// If Env is nil, use the current process' environment.
	cmd.Env = c.Env
	cmd.Dir = c.Dir

	// /////////////// Start command
	now := time.Now()
	if err := cmd.Start(); err != nil {
		c.Lock()
		c.status.Error = err
		c.status.StartTs = now.UnixNano()
		c.status.StopTs = time.Now().UnixNano()
		c.done = true
		c.Unlock()
		return
	}

	// Set initial status
	c.Lock()
	c.startTime = now              // command is running
	c.status.PID = cmd.Process.Pid // command is running
	c.status.StartTs = now.UnixNano()
	c.started = true
	c.Unlock()

	//  /////////////// Wait for command to finish or be killed
	err := cmd.Wait()
	now = time.Now()

	// Get exit code of the command. According to the manual, Wait() returns:
	// "If the command fails to run or doesn't complete successfully, the error
	// is of type *ExitError. Other error types may be returned for I/O problems."
	exitCode := 0
	signaled := false
	if err != nil {
		if errt, ok := err.(*exec.ExitError); ok {
			// This is the normal case which is not really an error. It's string
			// representation is only "*exec.ExitError". It only means the cmd
			// did not exit zero and caller should see ExitError.Stderr, which
			// we already have. So first we'll have this as the real/underlying
			// type, then discard err so status.Error doesn't contain a useless
			// "*exec.ExitError". With the real type we can get the non-zero
			// exit code and determine if the process was signaled, which yields
			// a more specific error message, so we set err again in that case.
			err = nil
			if waitStatus, ok := errt.Sys().(syscall.WaitStatus); ok {
				exitCode = waitStatus.ExitStatus() // -1 if signaled
				if waitStatus.Signaled() {
					signaled = true
					err = errors.New(errt.Error()) // "signal: terminated"
				}
			}
		}
	}

	// Set final status
	c.Lock()
	if !c.stopped && !signaled {
		c.status.Complete = true
	}
	c.status.Runtime = now.Sub(c.startTime).Seconds()
	c.status.StopTs = now.UnixNano()
	c.status.Exit = exitCode
	c.status.Error = err
	c.done = true
	c.Unlock()
}
