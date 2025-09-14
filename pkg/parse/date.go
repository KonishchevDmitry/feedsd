package parse

import (
	"fmt"
	"regexp"
	"time"
)

var russianDateRegex = regexp.MustCompile(`^(\d{1,2}) ([а-я]+\.?) (\d{4})$`)

func Date(value string) (time.Time, error) {
	matches := russianDateRegex.FindStringSubmatch(value)
	if len(matches) == 0 {
		return time.Time{}, fmt.Errorf("invalid date: %q", value)
	}

	month, err := parseMonth(matches[2])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date: %q", value)
	}

	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to load timezone: %w", err)
	}

	date, err := time.ParseInLocation("2.1.2006", fmt.Sprintf("%s.%d.%s", matches[1], month, matches[3]), location)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date: %q", value)
	}

	return date, nil
}

func parseMonth(month string) (int, error) {
	switch month {
	case "января", "янв.":
		return 1, nil
	case "февраля", "февр.":
		return 2, nil
	case "марта", "мар.":
		return 3, nil
	case "апреля", "апр.":
		return 4, nil
	case "мая":
		return 5, nil
	case "июня", "июн.":
		return 6, nil
	case "июля", "июл.":
		return 7, nil
	case "августа", "авг.":
		return 8, nil
	case "сентября", "сен.":
		return 9, nil
	case "октября", "окт.":
		return 10, nil
	case "ноября", "нояб.":
		return 11, nil
	case "декабря", "дек.":
		return 12, nil
	default:
		return 0, fmt.Errorf("invalid month: %q", month)
	}
}
