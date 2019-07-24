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
		cmd.Buffered(false), cmd.Streaming(false))
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
