package query

import (
	"fmt"

	"github.com/PuerkitoBio/goquery"
)

func Optional(selection *goquery.Selection, name string, selector string) (*goquery.Selection, bool, error) {
	selection = selection.Find(selector)

	switch size := selection.Size(); size {
	case 0:
		return nil, false, nil
	case 1:
		return selection, true, nil
	default:
		return nil, false, fmt.Errorf("unable to find %s: got %d elements that match %q selector", name, size, selector)
	}
}

func One(selection *goquery.Selection, name string, selector string) (*goquery.Selection, error) {
	selection = selection.Find(selector)

	switch size := selection.Size(); size {
	case 0:
		return nil, fmt.Errorf("unable to find %s", name)
	case 1:
		return selection, nil
	default:
		return nil, fmt.Errorf("unable to find %s: got %d elements that match %q selector", name, size, selector)
	}
}

func Many(selection *goquery.Selection, name string, selector string) (*goquery.Selection, error) {
	selection = selection.Find(selector)
	if selection.Size() == 0 {
		return nil, fmt.Errorf("unable to find %s", name)
	}
	return selection, nil
}

func ForEach(selection *goquery.Selection, process func(selection *goquery.Selection) error) error {
	var err error
	selection.EachWithBreak(func(i int, selection *goquery.Selection) bool {
		err = process(selection)
		return err == nil
	})
	return err
}

func Map[T any](selection *goquery.Selection, mapper func(*goquery.Selection) (T, error)) ([]T, error) {
	var items []T

	if err := ForEach(selection, func(selection *goquery.Selection) error {
		item, err := mapper(selection)
		if err == nil {
			items = append(items, item)
		}
		return err
	}); err != nil {
		return nil, err
	}

	return items, nil
}
