package lyrics

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// lrcPattern matches standard LRC timestamps: [mm:ss.xx] or [mm:ss]
var lrcPattern = regexp.MustCompile(`\[(\d{1,3}):(\d{1,2})(?:\.(\d{1,3}))?\]`)

// ParseLRC parses a standard LRC (LyRiCs) formatted string into LyricLines.
func ParseLRC(lrcText string) ([]LyricLine, error) {
	if lrcText == "" {
		return nil, fmt.Errorf("empty LRC text")
	}

	var lines []LyricLine

	for _, raw := range strings.Split(lrcText, "\n") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		// Find all timestamps in the line
		matches := lrcPattern.FindAllStringSubmatch(raw, -1)
		if len(matches) == 0 {
			continue
		}

		// Extract text after all timestamps
		text := lrcPattern.ReplaceAllString(raw, "")
		text = strings.TrimSpace(text)

		for _, match := range matches {
			if len(match) < 3 {
				continue
			}
			minutes, err := strconv.Atoi(match[1])
			if err != nil {
				continue
			}
			seconds, err := strconv.Atoi(match[2])
			if err != nil {
				continue
			}
			millis := 0
			if len(match) == 4 && match[3] != "" {
				ms, err := strconv.Atoi(match[3])
				if err == nil {
					// Handle both 2-digit and 3-digit milliseconds
					if ms < 100 {
						ms *= 10
					}
					millis = ms
				}
			}

			timeMs := minutes*60*1000 + seconds*1000 + millis

			lines = append(lines, LyricLine{
				TimeMs: timeMs,
				Text:   text,
			})
		}
	}

	if len(lines) == 0 {
		return nil, fmt.Errorf("no valid LRC timestamps found")
	}

	return lines, nil
}
