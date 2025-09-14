package parse

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrimText(t *testing.T) {
	const nbsp = "\u00a0"
	const softHypen = "\u00ad"

	require.Equal(t, "some text with hypened word", TrimText(fmt.Sprintf(
		" \t\nsome%s text with hype%sned word \r\n", nbsp, softHypen,
	)))
}
