package cmd

import (
	"sync"
	"time"
)

// Cmd represents an external command, similar to the Go built-in os/exec.Cmd.
// A Cmd cannot be reused after calling Start. Exported fields are read-only and
// should not be modified, except Env which can be set before calling Start.
// To create a new Cmd, call NewCmd or NewCmdOptions.
type Cmd struct {
	Name string
	Args []string
	Env  []string
	Dir  string

	Stdout     chan string   // streaming STDOUT if enabled, else nil (see Options)
	Stderr     chan string   // streaming STDERR if enabled, else nil (see Options)
	statusChan chan Status   // nil until Start() called
	doneChan   chan struct{} // closed when done running

	*sync.Mutex

	started  bool // cmd.Start called, no error
	stopped  bool // Stop called
	done     bool // run() done
	final    bool // status finalized in Status
	buffered bool // buffer STDOUT and STDERR to Status.Stdout and Std

	startTime time.Time     // if started true
	stdout    *OutputBuffer // low-level stdout buffering and streaming
	stderr    *OutputBuffer // low-level stderr buffering and streaming
	status    Status
	Timeout   time.Duration
}

// Status represents the running status and consolidated return of a Cmd. It can
// be obtained any time by calling Cmd.Status. If StartTs > 0, the command has
// started. If StopTs > 0, the command has stopped. After the command finishes
// for any reason, this combination of values indicates success (presuming the
// command only exits zero on success):
//
//   Exit     = 0
//   Error    = nil
//   Complete = true
//
// Error is a Go error from the underlying os/exec.Cmd.Start or os/exec.Cmd.Wait.
// If not nil, the command either failed to start (it never ran) or it started
// but was terminated unexpectedly (probably signaled). In either case, the
// command failed. Callers should check Error first. If nil, then check Exit and
// Complete.
type Status struct {
	Cmd      string
	PID      int
	Complete bool     // false if stopped or signaled
	Exit     int      // exit code of process
	Error    error    // Go error
	StartTs  int64    // Unix ts (nanoseconds), zero if Cmd not started
	StopTs   int64    // Unix ts (nanoseconds), zero if Cmd not started or running
	Runtime  float64  // seconds, zero if Cmd not started
	Stdout   []string // buffered STDOUT; see Cmd.Status for more info
	Stderr   []string // buffered STDERR; see Cmd.Status for more info
}

// Options represents customizations for NewCmdOptions.
type Options struct {
	// If Buffered is true, STDOUT and STDERR are written to Status.Stdout and
	// Status.Stderr. The caller can call Cmd.Status to read output at intervals.
	// See Cmd.Status for more info.
	Buffered bool

	// If Streaming is true, Cmd.Stdout and Cmd.Stderr channels are created and
	// STDOUT and STDERR output lines are written them in real time. This is
	// faster and more efficient than polling Cmd.Status. The caller must read both
	// streaming channels, else lines are dropped silently.
	Streaming bool

	// Set timeout for execution
	Timeout time.Duration
}
