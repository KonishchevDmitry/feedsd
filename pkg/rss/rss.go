package rss

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"

	"golang.org/x/net/html/charset"
)

const ContentType = "application/rss+xml"

var PossibleContentTypes = []string{ContentType, "application/xml", "text/xml"}

func Read(reader io.Reader, ignoreCharset bool) (*Feed, error) {
	rss := rssRoot{}

	decoder := xml.NewDecoder(reader)
	decoder.Strict = false
	decoder.CharsetReader = func(encoding string, input io.Reader) (io.Reader, error) {
		if ignoreCharset {
			return input, nil
		}
		return charset.NewReaderLabel(encoding, input)
	}

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
	return Read(bytes.NewReader(data), false)
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

func Generate(feed *Feed) ([]byte, error) {
	var buffer bytes.Buffer
	if err := Write(feed, &buffer); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
