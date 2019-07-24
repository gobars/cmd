package cmd

import "fmt"

// --------------------------------------------------------------------------

const (
	// DEFAULT_LINE_BUFFER_SIZE is the default size of the OutputStream line buffer.
	// The default value is usually sufficient, but if ErrLineBufferOverflow errors
	// occur, try increasing the size by calling OutputBuffer.SetLineBufferSize.
	DefaultLineBufferSize = 16384

	// DEFAULT_STREAM_CHAN_SIZE is the default string channel size for a Cmd when
	// Options.Streaming is true. The string channel size can have a minor
	// performance impact if too small by causing OutputStream.Write to block
	// excessively.
	DefaultStreamChanSize = 1000
)

// ErrLineBufferOverflow is returned by OutputStream.Write when the internal
// line buffer is filled before a newline character is written to terminate a
// line. Increasing the line buffer size by calling OutputStream.SetLineBufferSize
// can help prevent this error.
type ErrLineBufferOverflow struct {
	Line       string // Unterminated line that caused the error
	BufferSize int    // Internal line buffer size
	BufferFree int    // Free bytes in line buffer
}

func (e ErrLineBufferOverflow) Error() string {
	return fmt.Sprintf("line does not contain newline and is %d bytes too long to buffer (buffer size: %d)",
		len(e.Line)-e.BufferSize, e.BufferSize)
}
