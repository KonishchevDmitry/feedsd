package rss

import (
	"os"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/require"
)

func TestParseMinimal(t *testing.T) {
	t.Parallel()
	testParse(t, heredoc.Doc(`
        <?xml version="1.0" encoding="UTF-8"?>
        <rss version="2.0">
            <channel>
                <title>Feed title</title>
                <link>http://example.com/</link>
                <description>Feed description</description>
            </channel>
        </rss>`,
	), "")
}

func TestParseFull(t *testing.T) {
	t.Parallel()
	testParse(t, heredoc.Doc(`
        <?xml version="1.0" encoding="UTF-8"?>
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
                    <encoded xmlns="http://purl.org/rss/1.0/modules/content/">Item 1 content</encoded>
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
        </rss>`,
	), "")
}

func TestParseMalformed(t *testing.T) {
	t.Parallel()
	// item.content payload is not escaped
	testParse(t, heredoc.Doc(`
        <rss xmlns:atom="http://www.w3.org/2005/Atom" xmlns:yandex="http://news.yandex.ru" version="2.0">
            <channel>
                <title>Forbes.ru</title>
                <link>https://www.forbes.ru</link>
                <item>
                    <title>Суд Флориды отклонил иск Трампа к The New York Times на $15 млрд</title>
                    <guid isPermaLink="true">forbes-546283</guid>
                    <link>https://www.forbes.ru/biznes/546283-sud-floridy-otklonil-isk-trampa-k-the-new-york-times-na-15-mlrd</link>
                    <description>
                        <![CDATA[ Суд Флориды отклонил иск Трампа к The New York Times на $15 млрд, сообщила USA Today. Судья объяснил свое решение тем, что в документе не было четко изложенной претензии для рассмотрения. Трампу дали 28 дней на подачу другой версии иска, которая должна быть вдвое короче. Президент США ранее обвинил газету в публикации «злонамеренных обвинений» и процитировал в иске несколько статей, включая редакционную колонку накануне выборов 2024 года, в которой утверждалось, что он непригоден для должности главы страны ]]>
                    </description>
                    <pubDate>Fri, 19 Sep 2025 22:14:03 +0300</pubDate>
                    <author>Евгения Белкова</author>
                    <category>Бизнес</category>
                    <content> <p>Суд Флориды отклонил иск президента США Дональда Трампа к газете The New York Times на $15 млрд, об этом <a href="https://www.usatoday.com/story/news/politics/2025/09/19/judge-trump-lawsuit-new-york-times/86241281007/">сообщила</a> USA Today. Судья объяснила решение тем, что в документе не было четко изложенной претензии для рассмотрения, однако Трампу дали еще 28 дней для подачи другой версии иска. «Как знает каждый юрист (или должен знать), иск — это не публичная трибуна для оскорблений и брани, не защищенная площадка для нападок на оппонента. Иск — это не мегафон для PR-кампании или трибуна для страстной речи на политическом митинге или функциональный эквивалент угла ораторов в Гайд-парке», — заявил судья Стивен Мерридей.&nbsp;</p> <p>USA Today добавила, что в иске Трамп цитировал серию статей газеты, включая редакционную колонку накануне выборов президента США 2024 года, в которой утверждалось, что он непригоден для должности главы страны, а также книгу, выпущенную издательством Penguin, — «Счастливый неудачник: как Дональд Трамп растратил состояние своего отца и&nbsp;создал иллюзию успеха». «Ответчики злонамеренно опубликовали книгу и статьи, зная, что эти публикации наполнены отвратительными искажениями и выдумками о президенте Трампе», — утверждалось в иске.&nbsp;</p> <p>Первоначально документ президента, направленный в суд, содержал 85 страниц, судья Мерридей ограничил новую версию 40 страницами, отметило СМИ. В качестве ответчиков по делу указаны журналисты Сюзанна Крейг, Росс Бюттнер, Питер Бейкер и Майкл Шмидт, а также сама газета The New York Times и издательство Penguin Books. Авторами упомянутой книги выступили Крейг и Бюттнер.</p> <p>The New York Times поприветствовала решение суда, добавила USA Today. Издание подчеркнуло, что иск Трампа «не имеет юридических оснований» и призван запугать независимую журналистику.&nbsp;</p> <p> Об иске Трампа к NYT на $15 млрд в начале недели <a href="https://www.forbes.ru/society/545948-tramp-podal-isk-o-klevete-na-15-mlrd-k-the-new-york-times-i-penguin">сообщило</a><a href="https://www.forbes.ru/society/545948-tramp-podal-isk-o-klevete-na-15-mlrd-k-the-new-york-times-i-penguin"></a> агентство Reuters. Глава New York Times Мередит Левиен&nbsp;несколько дней спустя <a href="https://www.forbes.ru/biznes/546146-glava-nyt-obvinila-trampa-v-davlenii-na-smi-posle-iska-na-15-mlrd">обвинила</a> президента Штатов в давлении на СМИ. Она назвала его иск безосновательным и назвала инцидент частью кампании по запугиванию независимых журналистов, проведя параллель с авторитарными тактиками в таких странах, как Турция и Венгрия.&nbsp;</p> <p>FT ранее писала, что с марта 2024 года Трамп подал иски на миллиарды долларов против четырех крупнейших медиа страны: ABC News, CBS News, The Wall Street Journal и NYT. В 2025 году ABC News и CBS News урегулировали претензии президента, согласившись выплатить в фонд его будущей президентской библиотеки $15 млн и $16 млн соответственно.&nbsp;</p> </content>
                </item>
            </channel>
        </rss>`,
	), heredoc.Doc(`
        <?xml version="1.0" encoding="UTF-8"?>
        <rss version="2.0">
            <channel>
                <title>Forbes.ru</title>
                <link>https://www.forbes.ru</link>
                <description></description>
                <item>
                    <title>Суд Флориды отклонил иск Трампа к The New York Times на $15 млрд</title>
                    <guid isPermaLink="true">forbes-546283</guid>
                    <link>https://www.forbes.ru/biznes/546283-sud-floridy-otklonil-isk-trampa-k-the-new-york-times-na-15-mlrd</link>
                    <description>&#xA;                 Суд Флориды отклонил иск Трампа к The New York Times на $15 млрд, сообщила USA Today. Судья объяснил свое решение тем, что в документе не было четко изложенной претензии для рассмотрения. Трампу дали 28 дней на подачу другой версии иска, которая должна быть вдвое короче. Президент США ранее обвинил газету в публикации «злонамеренных обвинений» и процитировал в иске несколько статей, включая редакционную колонку накануне выборов 2024 года, в которой утверждалось, что он непригоден для должности главы страны &#xA;            </description>
                    <pubDate>Fri, 19 Sep 2025 19:14:03 GMT</pubDate>
                    <author>Евгения Белкова</author>
                    <category>Бизнес</category>
                </item>
            </channel>
        </rss>`,
	))
}

func TestReadRss091WithCustomEncoding(t *testing.T) {
	t.Parallel()

	file, err := os.Open("testdata/rss-0.91-with-custom-encoding.xml")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, file.Close())
	}()

	feed, err := Read(file, false)
	require.NoError(t, err)

	require.Equal(t, "Свежачок от LostFilm.TV", feed.Description)
	require.Equal(t, "Непокорная Земля (Defiance). Мир, который мы захватим/Последние единороги (The World We Seize/The Last Unicorns) [MP4]. (S03E01-2)", feed.Items[0].Title)
}

func testParse(t *testing.T, data string, expected string) {
	if expected == "" {
		expected = data
	}

	feed, err := Parse([]byte(data))
	require.NoError(t, err)

	generatedData, err := Generate(feed)
	require.NoError(t, err)

	require.Equal(t, expected, string(generatedData))
}
