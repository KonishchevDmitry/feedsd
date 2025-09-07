package rss

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"

	"golang.org/x/net/html/charset"
)

func Read(reader io.Reader) (*Feed, error) {
	rss := rssRoot{}

	decoder := xml.NewDecoder(reader)
	decoder.CharsetReader = charset.NewReaderLabel

	if err := decoder.Decode(&rss); err != nil {
		return nil, err
	}

	switch rss.Version {
	case "2.0", "0.92", "0.91":
	default:
		return nil, fmt.Errorf("unsupported RSS version: %s", rss.Version)
	}

	if rss.Channel == nil {
		return nil, errors.New("the document doesn't conform to RSS specification")
	}

	return rss.Channel, nil
}

func Parse(data []byte) (*Feed, error) {
	return Read(bytes.NewReader(data))
}

func Write(feed *Feed, writer io.Writer) error {
	if _, err := writer.Write([]byte(xml.Header)); err != nil {
		return err
	}

	rss := rssRoot{Version: "2.0", Channel: feed}
	encoder := xml.NewEncoder(writer)
	encoder.Indent("", "    ")
	return encoder.Encode(&rss)
}

func Generate(origFeed *Feed, postprocess bool) ([]byte, error) {
	feed := *origFeed

	if postprocess {
		feed.Items = nil
		trueValue := true

		for _, item := range origFeed.Items {
			item := *item

			if guid := &item.GUID; guid.ID == "" && item.Link != "" {
				guid.ID = item.Link
				guid.IsPermaLink = &trueValue
			}

			feed.Items = append(feed.Items, &item)
		}
	}

	var buffer bytes.Buffer
	if err := Write(&feed, &buffer); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
