package cmd_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/bingoohuang/cmd"
	"github.com/stretchr/testify/assert"
)

func TestBash(t *testing.T) {
	_, status := cmd.Bash(`echo "Hello"`, cmd.Timeout(1*time.Second))
	assert.Equal(t, []string{"Hello"}, status.Stdout)
}

func TestBashBufferedOff(t *testing.T) {
	_, status := cmd.Bash(`echo "Hello"`, cmd.Timeout(1*time.Second),
		cmd.Buffered(false))
	assert.Nil(t, status.Stdout)
}

func TestBashLinerFalse(t *testing.T) {
	_, status := cmd.BashLiner(`echo hello; sleep 2; echo world;`, func(line string) bool {
		fmt.Println(line)
		return false
	}, cmd.Timeout(3*time.Second))
	assert.NotNil(t, status.Error)
}

func TestBashLinerTrue(t *testing.T) {
	_, status := cmd.BashLiner(`echo hello; sleep 2; echo world;`, func(line string) bool {
		fmt.Println(line)
		return true
	}, cmd.Timeout(1*time.Second))
	assert.NotNil(t, status.Error)
}

func TestStdinEnabled(t *testing.T) {
	p := cmd.NewCmd("bash", "-c", "cat")
	p.Options(cmd.Stdin())
	chanStatuses := p.Start()

	p.Stdin <- "Input string"
	p.Stdin <- "Line 2"
	close(p.Stdin)

	err := p.Stop()
	assert.Nil(t, err)

	status := <-chanStatuses

	assert.Equal(t, []string{"Input string", "Line 2"}, status.Stdout)
	_ = p.Stop()
}

func TestStdinEnabledStream(t *testing.T) {
	p := cmd.NewCmd("bash", "-c", "cat")
	p.Options(cmd.Stdin(), cmd.Streaming(), cmd.Buffered(false))
	p.Start()

	p.Stdin <- "Line 1"
	line := <-p.Stdout
	assert.Equal(t, "Line 1", line)
	p.Stdin <- "Line 2"
	line = <-p.Stdout
	assert.Equal(t, "Line 2", line)
	close(p.Stdin)

	_ = p.Stop()
}
