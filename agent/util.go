package agent

import (
	"log"
	"strings"
	"time"
)

const (
	dateSearchFormat = "1/2/2006,03:04:05 PM"
	dateFormat       = "1/2/2006 15:04:05"
)

func parseDate(dt string) time.Time {
	t, err := time.Parse(dateFormat, dt)
	if err != nil {
		log.Printf("parseDate: %s could not be parsed, using now()", dt)
		return time.Now()
	}
	return t
}

// unwantedTraffic determines if a URL should be stored in memory or not
func unwantedTraffic(url string) bool {
	return !strings.HasPrefix(url, "http") ||
		strings.HasSuffix(url, ".css") ||
		strings.HasSuffix(url, ".js") ||
		strings.HasSuffix(url, ".svg") ||
		strings.HasSuffix(url, ".woff2")
}
