package cmd

import (
	"bufio"
	"bytes"
	"sync"
)

// //////////////////////////////////////////////////////////////////////////
// Output
// //////////////////////////////////////////////////////////////////////////

// os/exec.Cmd.StdoutPipe is usually used incorrectly. The docs are clear:
// "it is incorrect to call Wait before all reads from the pipe have completed."
// Therefore, we can't read from the pipe in another goroutine because it
// causes a race condition: we'll read in one goroutine and the original
// goroutine that calls Wait will write on close which is what Wait does.
// The proper solution is using an io.Writer for cmd.Stdout. I couldn't find
// an io.Writer that's also safe for concurrent reads (as lines in a []string
// no less), so I created one:

// OutputBuffer represents command output that is saved, line by line, in an
// unbounded buffer. It is safe for multiple goroutines to read while the command
// is running and after it has finished. If output is small (a few megabytes)
// and not read frequently, an output buffer is a good solution.
//
// A Cmd in this package uses an OutputBuffer for both STDOUT and STDERR by
// default when created by calling NewCmd. To use OutputBuffer directly with
// a Go standard library os/exec.Command:
//
//   import "os/exec"
//   import "github.com/bingoohuang/cmd"
//   runnableCmd := exec.Command(...)
//   stdout := cmd.NewOutputBuffer()
//   runnableCmd.Stdout = stdout
//
// While runnableCmd is running, call stdout.Lines() to read all output
// currently written.
type OutputBuffer struct {
	buf   *bytes.Buffer
	lines []string
	*sync.Mutex
}

// NewOutputBuffer creates a new output buffer. The buffer is unbounded and safe
// for multiple goroutines to read while the command is running by calling Lines.
func NewOutputBuffer() *OutputBuffer {
	out := &OutputBuffer{
		buf:   &bytes.Buffer{},
		lines: []string{},
		Mutex: &sync.Mutex{},
	}
	return out
}

// Write makes OutputBuffer implement the io.Writer interface. Do not call
// this function directly.
func (rw *OutputBuffer) Write(p []byte) (n int, err error) {
	rw.Lock()
	defer rw.Unlock()
	return rw.buf.Write(p) // and bytes.Buffer implements io.Writer
}

// Lines returns lines of output written by the Cmd. It is safe to call while
// the Cmd is running and after it has finished. Subsequent calls returns more
// lines, if more lines were written. "\r\n" are stripped from the lines.
func (rw *OutputBuffer) Lines() []string {
	rw.Lock()
	defer rw.Unlock()
	// Scanners are io.Readers which effectively destroy the buffer by reading
	// to EOF. So once we scan the buf to lines, the buf is empty again.
	s := bufio.NewScanner(rw.buf)
	for s.Scan() {
		rw.lines = append(rw.lines, s.Text())
	}
	return rw.lines
}
