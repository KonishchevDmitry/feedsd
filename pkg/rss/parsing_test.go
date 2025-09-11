package rss

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var minimalRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
    <channel>
        <title>Feed title</title>
        <link>http://example.com/</link>
        <description>Feed description</description>
    </channel>
</rss>`

var fullRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
    <channel>
        <title>Feed title</title>
        <link>http://example.com/</link>
        <description>Feed description</description>
        <image>
            <url>http://example.com/logo.png</url>
            <title>Logo title</title>
            <link>http://example.com/</link>
            <width>100</width>
        </image>
        <language>en-us</language>
        <pubDate>Sat, 04 Apr 2015 00:00:00 GMT</pubDate>
        <category>feed-cat1</category>
        <category>feed-cat2</category>
        <generator>go-rss</generator>
        <ttl>60</ttl>
        <item>
            <title>Item 1</title>
            <guid isPermaLink="true">http://example.com/item1</guid>
            <link>http://example.com/item1</link>
            <description>Item 1 description</description>
            <enclosure url="http://example.com/item1/podcast.mp3" type="audio/mpeg" length="123456789"></enclosure>
            <comments>http://example.com/item1/comments</comments>
            <pubDate>Sat, 04 Apr 2015 07:00:13 GMT</pubDate>
            <author>author1</author>
            <category>item-cat1</category>
            <category>item-cat2</category>
        </item>
        <item>
            <title>Item 2</title>
            <guid isPermaLink="false">2e17b013-f283-45e4-b010-5a03ad6776c6</guid>
        </item>
        <item>
            <title>Item 3</title>
            <guid>http://example.com/item3</guid>
        </item>
        <item></item>
        <item>
            <title>Охотники за привидениями - Русский Трейлер (2016)</title>
            <link>http://www.youtube.com/watch?v=jhduECOtxPI</link>
            <group xmlns="http://search.yahoo.com/mrss/">
                <title>Охотники за привидениями - Русский Трейлер (2016)</title>
                <thumbnail url="https://i3.ytimg.com/vi/jhduECOtxPI/hqdefault.jpg" width="480" height="360"></thumbnail>
                <content url="https://www.youtube.com/v/jhduECOtxPI?version=3" type="application/x-shockwave-flash" width="640" height="390"></content>
                <description>Официальный русский трейлер фильма Охотники за привидениями (2016)</description>
            </group>
        </item>
    </channel>
</rss>`

func TestParseMinimal(t *testing.T) {
	t.Parallel()
	testParse(t, minimalRSS)
}

func TestParseFull(t *testing.T) {
	t.Parallel()
	testParse(t, fullRSS)
}

func TestReadRss091WithCustomEncoding(t *testing.T) {
	t.Parallel()

	file, err := os.Open("testdata/rss-0.91-with-custom-encoding.xml")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, file.Close())
	}()

	feed, err := Read(file)
	require.NoError(t, err)

	require.Equal(t, "Свежачок от LostFilm.TV", feed.Description)
	require.Equal(t, "Непокорная Земля (Defiance). Мир, который мы захватим/Последние единороги (The World We Seize/The Last Unicorns) [MP4]. (S03E01-2)", feed.Items[0].Title)
}

func testParse(t *testing.T, data string) {
	feed, err := Parse([]byte(data))
	require.NoError(t, err)

	generatedData, err := Generate(feed)
	require.NoError(t, err)

	require.Equal(t, data, string(generatedData))
}
