package rss

import (
	"fmt"
	"html"
	"net/url"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/samber/mo"

	"github.com/KonishchevDmitry/feedsd/pkg/filter"
)

type Feed struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	Description string   `xml:"description"`
	Image       *Image   `xml:"image"`
	Language    string   `xml:"language,omitempty"`
	Date        Date     `xml:"pubDate"`
	Category    []string `xml:"category"`
	Generator   string   `xml:"generator,omitempty"`
	TTL         int      `xml:"ttl,omitempty"`
	Items       []*Item  `xml:"item"`
}

func NewFeed(title string, link *url.URL) *Feed {
	return &Feed{
		Title: title,
		Link:  link.String(),
	}
}

func (f *Feed) AddItem(time time.Time, title string, link *url.URL, description string) {
	f.Items = append(f.Items, NewItem(time, title, link, description))
}

func (f *Feed) Filter(filter func(item *Item) bool) {
	f.Items = slices.DeleteFunc(f.Items, func(item *Item) bool {
		return !filter(item)
	})
}

func (f *Feed) BlockCategories(blacklist filter.Blacklist) {
	f.Filter(func(item *Item) bool {
		for _, category := range item.Categories {
			if blacklist.IsBlacklisted(category) {
				return false
			}
		}
		return true
	})
}

func (f *Feed) Deduplicate() {
	var (
		count int
		ids   = make(map[string]struct{})
	)

	for _, item := range f.Items {
		var id mo.Option[string]
		if guid := item.GUID.ID; guid != "" {
			id = mo.Some(guid)
		} else if link := item.Link; link != "" {
			id = mo.Some(link)
		}

		if id, ok := id.Get(); ok {
			if _, ok := ids[id]; ok {
				continue
			}
			ids[id] = struct{}{}
		}

		f.Items[count] = item
		count++
	}

	f.Items = f.Items[:count]
	sort.Slice(f.Items, func(i, j int) bool {
		return f.Items[i].Date.After(f.Items[j].Date.Time)
	})
}

func (f *Feed) Normalize() {
	trueValue := true

	for _, item := range f.Items {
		if guid := &item.GUID; guid.ID == "" && item.Link != "" {
			guid.ID = item.Link
			guid.IsPermaLink = &trueValue
		}
	}
}

func (f *Feed) String() string {
	if f == nil {
		return fmt.Sprintf("%#v", f)
	}

	xml, err := Generate(f)
	if err == nil {
		return string(xml)
	}

	return fmt.Sprintf("XML generation error: %s. Go representation: %#v", err, f)
}

type Image struct {
	URL    string `xml:"url"`
	Title  string `xml:"title"`
	Link   string `xml:"link"`
	Width  int    `xml:"width,omitempty"`
	Height int    `xml:"height,omitempty"`
}

type Date struct {
	time.Time
}

var NoTime time.Time

type Item struct {
	Title        string          `xml:"title,omitempty"`
	GUID         GUID            `xml:"guid"`
	Link         string          `xml:"link,omitempty"`
	Description  string          `xml:"description,omitempty"`
	Enclosure    []*Enclosure    `xml:"enclosure"`
	MediaContent []*MediaContent `xml:"http://search.yahoo.com/mrss/ content"`
	MediaGroup   []*MediaGroup   `xml:"http://search.yahoo.com/mrss/ group"`
	Comments     string          `xml:"comments,omitempty"`
	Date         Date            `xml:"pubDate"`
	Author       string          `xml:"author,omitempty"`
	Categories   []string        `xml:"category"`
}

func NewItem(time time.Time, title string, link *url.URL, description string) *Item {
	return &Item{
		Title:       title,
		Link:        link.String(),
		Description: description,
		Date:        Date{Time: time},
	}
}

func (i *Item) AddCategoriesToDescription() {
	if len(i.Categories) != 0 {
		i.Description += fmt.Sprintf("<p>%s</p>", html.EscapeString(strings.Join(i.Categories, " | ")))
	}
}

type GUID struct {
	ID          string `xml:",chardata"`
	IsPermaLink *bool  `xml:"isPermaLink,attr,omitempty"`
}

func MakeGUID(id string, isPermaLink bool) GUID {
	guid := GUID{ID: id}
	if !isPermaLink {
		guid.IsPermaLink = &isPermaLink
	}
	return guid
}

type Enclosure struct {
	URL    string `xml:"url,attr"`
	Type   string `xml:"type,attr"`
	Length int    `xml:"length,attr"`
}

type MediaGroup struct {
	Title       *MediaDescription `xml:"title"`
	Thumbnail   *MediaThumbnail   `xml:"thumbnail"`
	Content     *MediaContent     `xml:"content"`
	Description *MediaDescription `xml:"description"`
	Keywords    string            `xml:"keywords,omitempty"`
}

type MediaContent struct {
	Title     *MediaDescription `xml:"title"`
	Thumbnail *MediaThumbnail   `xml:"thumbnail"`

	URL        string `xml:"url,attr,omitempty"`
	Medium     string `xml:"medium,attr,omitempty"`
	Type       string `xml:"type,attr,omitempty"`
	Expression string `xml:"expression,attr,omitempty"`
	Width      int    `xml:"width,attr,omitempty"`
	Height     int    `xml:"height,attr,omitempty"`
	IsDefault  bool   `xml:"isDefault,attr,omitempty"`

	Description *MediaDescription `xml:"description"`
	Keywords    string            `xml:"keywords,omitempty"`
}

type MediaDescription struct {
	Text string `xml:",chardata"`
	Type string `xml:"type,attr,omitempty"`
}

type MediaThumbnail struct {
	URL    string `xml:"url,attr,omitempty"`
	Width  int    `xml:"width,attr,omitempty"`
	Height int    `xml:"height,attr,omitempty"`
}
