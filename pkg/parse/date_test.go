package parse

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDate(t *testing.T) {
	t.Parallel()

	location, err := time.LoadLocation("Europe/Moscow")
	require.NoError(t, err)

	date, err := Date("12 сентября 2025")
	require.NoError(t, err)
	require.Equal(t, time.Date(2025, 9, 12, 0, 0, 0, 0, location), date)
}
