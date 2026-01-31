package parse

import (
	"fmt"
	"testing"

	"github.com/MakeNowJust/heredoc"

	"github.com/stretchr/testify/require"
)

func TestTrimText(t *testing.T) {
	const nbsp = "\u00a0"
	const softHypen = "\u00ad"

	require.Equal(t, "some text with hypened word", TrimText(fmt.Sprintf(
		" \t\nsome%s text with hype%sned word \r\n", nbsp, softHypen,
	)))
}

func TestTextToHTML(t *testing.T) {
	require.Equal(t, heredoc.Doc(`
		Some sentence.<br/>
		<br/>
		Simple links: <a href="https://github.com/KonishchevDmitry/">https://github.com/KonishchevDmitry/</a> <a href="https://github.com/KonishchevDmitry/feedsd/">https://github.com/KonishchevDmitry/feedsd/</a><br/>
		Links with punctuation: (<a href="https://github.com/">https://github.com/</a>), <a href="https://github.com/KonishchevDmitry/">https://github.com/KonishchevDmitry/</a>, <a href="https://github.com/KonishchevDmitry/feedsd">https://github.com/KonishchevDmitry/feedsd</a>.<br/>
		Complex link: <a href="https://example.com/resource?arg=value&amp;arg=value#param">https://example.com/resource?arg=value&amp;arg=value#param</a>?!<br/>
		<br/>
		HTML-escaped text: &lt;b&gt;&#39;escaped&#34;&lt;/b&gt;.<br/>
	`), TextToHTML(heredoc.Doc(`
		Some sentence.

		Simple links: https://github.com/KonishchevDmitry/ https://github.com/KonishchevDmitry/feedsd/
		Links with punctuation: (https://github.com/), https://github.com/KonishchevDmitry/, https://github.com/KonishchevDmitry/feedsd.
		Complex link: https://example.com/resource?arg=value&arg=value#param?!

		HTML-escaped text: <b>'escaped"</b>.
	`)))
}
