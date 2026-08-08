package makeme

import (
	"fmt"
	"time"
)

func ptr[T any](value T) *T {
	return &value
}

func fixtureCode(prefix string, sequence int64, width int) string {
	return fmt.Sprintf("%s%0*d", prefix, width, sequence)
}

func fixtureThaiText(prefix string, sequence int64) string {
	return fmt.Sprintf("%s %d", prefix, sequence)
}

func timeForSequence(sequence int64) time.Time {
	location := time.FixedZone("Asia/Bangkok", 7*60*60)
	return time.Date(2026, time.January, 2, 3, 4, 5, 0, location).Add(time.Duration(sequence) * time.Second)
}
