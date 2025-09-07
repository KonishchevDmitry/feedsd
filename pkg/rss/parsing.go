package rss

import (
	"encoding/xml"
	"fmt"
	"time"
)

type rssRoot struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel *Feed    `xml:"channel"`
}

func (g *GUID) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if g.ID == "" {
		return nil
	}

	if g.IsPermaLink != nil {
		value := "true"
		if !*g.IsPermaLink {
			value = "false"
		}

		attr := xml.Attr{
			Name:  xml.Name{Local: "isPermaLink"},
			Value: value,
		}

		start.Attr = append(start.Attr, attr)
	}

	if err := e.EncodeToken(start); err != nil {
		return err
	}

	if err := e.EncodeToken(xml.CharData(g.ID)); err != nil {
		return err
	}

	if err := e.EncodeToken(xml.EndElement{Name: start.Name}); err != nil {
		return err
	}

	return nil
}

func (d *Date) MarshalXML(encoder *xml.Encoder, start xml.StartElement) error {
	if d.IsZero() {
		return nil
	}

	if err := encoder.EncodeToken(start); err != nil {
		return err
	}

	if err := encoder.EncodeToken(xml.CharData(d.UTC().Format("Mon, 02 Jan 2006 15:04:05") + " GMT")); err != nil {
		return err
	}

	if err := encoder.EncodeToken(xml.EndElement{Name: start.Name}); err != nil {
		return err
	}

	return nil
}

func (d *Date) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) error {
	var value string
	if err := decoder.DecodeElement(&value, &start); err != nil {
		return err
	}

	for _, tz := range []string{"MST", "-0700"} {
		for _, year := range []string{"2006", "06"} {
			for _, day := range []string{"02", "2"} {
				for _, dayOfWeek := range []string{"Mon, ", ""} {
					format := fmt.Sprintf("%s%s Jan %s 15:04:05 %s", dayOfWeek, day, year, tz)
					if d.tryParse(format, value) {
						return nil
					}
				}
			}
		}
	}

	for _, format := range []string{
		"2006-01-02 15:04:05 -0700",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05.000-07:00",
	} {
		if d.tryParse(format, value) {
			return nil
		}
	}

	return fmt.Errorf("can't parse date: %s", value)
}

func (d *Date) tryParse(format string, value string) bool {
	var err error
	d.Time, err = time.Parse(format, value)
	return err == nil
}
