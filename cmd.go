// Package cmd runs external commands with concurrent access to output and
// status. It wraps the Go standard library os/exec.Command to correctly handle
// reading output (STDOUT and STDERR) while a command is running and killing a
// command. All operations are safe to call from multiple goroutines.
//
// A basic example that runs env and prints its output:
//
//   import (
//       "fmt"
//       "github.com/bingoohuang/cmd"
//   )
//
//   func main() {
//       // Create Cmd, buffered output
//       envCmd := cmd.NewCmd("env")
//
//       // Run and wait for Cmd to return Status
//       status := <-envCmd.Start()
//
//       // Print each line of STDOUT from Cmd
//       for _, line := range status.Stdout {
//           fmt.Println(line)
//       }
//   }
//
// Commands can be ran synchronously (blocking) or asynchronously (non-blocking):
//
//   envCmd := cmd.NewCmd("env") // create
//
//   status := <-envCmd.Start() // run blocking
//
//   statusChan := envCmd.Start() // run non-blocking
//   // Do other work while Cmd is running...
//   status <- statusChan // blocking
//
// Start returns a channel to which the final Status is sent when the command
// finishes for any reason. The first example blocks receiving on the channel.
// The second example is non-blocking because it saves the channel and receives
// on it later. Only one final status is sent to the channel; use Done for
// multiple goroutines to wait for the command to finish, then call Status to
// get the final status.
package cmd

import (
	"syscall"
	"time"
)

// Start starts the command and immediately returns a channel that the caller
// can use to receive the final Status of the command when it ends. The caller
// can start the command and wait like,
//
//   status := <-myCmd.Start() // blocking
//
// or start the command asynchronously and be notified later when it ends,
//
//   statusChan := myCmd.Start() // non-blocking
//   // Do other work while Cmd is running...
//   status := <-statusChan // blocking
//
// Exactly one Status is sent on the channel when the command ends. The channel
// is not closed. Any Go error is set to Status.Error. Start is idempotent; it
// always returns the same channel.
func (c *Cmd) Start() <-chan Status {
	c.Lock()
	defer c.Unlock()

	if c.statusChan != nil {
		return c.statusChan
	}

	c.statusChan = make(chan Status, 1)
	go c.run()
	return c.statusChan
}

// Stop stops the command by sending its process group a SIGTERM signal.
// Stop is idempotent. An error should only be returned in the rare case that
// Stop is called immediately after the command ends but before Start can
// update its internal state.
func (c *Cmd) Stop() error {
	c.Lock()
	defer c.Unlock()

	// Nothing to stop if Start hasn't been called, or the proc hasn't started,
	// or it's already done.
	if c.statusChan == nil || !c.started || c.done {
		return nil
	}

	// Flag that command was stopped, it didn't complete. This results in
	// status.Complete = false
	c.stopped = true

	// Signal the process group (-pid), not just the process, so that the process
	// and all its children are signaled. Else, child procs can keep running and
	// keep the stdout/stderr fd open and cause cmd.Wait to hang.
	return syscall.Kill(-c.status.PID, syscall.SIGTERM)
}

// Status returns the Status of the command at any time. It is safe to call
// concurrently by multiple goroutines.
//
// With buffered output, Status.Stdout and Status.Stderr contain the full output
// as of the Status call time. For example, if the command counts to 3 and three
// calls are made between counts, Status.Stdout contains:
//
//   "1"
//   "1 2"
//   "1 2 3"
//
// The caller is responsible for tailing the buffered output if needed. Else,
// consider using streaming output. When the command finishes, buffered output
// is complete and final.
//
// Status.Runtime is updated while the command is running and final when it
// finishes.
func (c *Cmd) Status() Status {
	c.Lock()
	defer c.Unlock()

	// Return default status if cmd hasn't been started
	if c.statusChan == nil || !c.started {
		return c.status
	}

	if c.done {
		// No longer running
		if !c.final {
			if c.buffered {
				c.status.Stdout = c.stdout.Lines()
				c.status.Stderr = c.stderr.Lines()
				c.stdout = nil // release buffers
				c.stderr = nil
			}
			c.final = true
		}
	} else {
		// Still running
		c.status.Runtime = time.Since(c.startTime).Seconds()
		if c.buffered {
			c.status.Stdout = c.stdout.Lines()
			c.status.Stderr = c.stderr.Lines()
		}
	}

	return c.status
}

// Done returns a channel that's closed when the command stops running.
// This method is useful for multiple goroutines to wait for the command
// to finish.Call Status after the command finishes to get its final status.
func (c *Cmd) Done() <-chan struct{} { return c.doneChan }
