package codec_test

import (
	"os"
	"testing"

	"github.com/echocat/cli4g"
)

func TestMain(m *testing.M) {
	cli.DefaultErrorHandler = nil
	os.Exit(m.Run())
}
