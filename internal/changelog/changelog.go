package changelog

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const (
	DefaultPath    = "changelog.md"
	ExpectedFormat = "# <version> - <summary>"
)

type Entry struct {
	Version     string
	Summary     string
	Description string
}

type ParseError struct {
	Path string
	Msg  string
	Err  error
}

func (e *ParseError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Path, e.Msg, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Msg)
}

func (e *ParseError) Unwrap() error { return e.Err }

func ParseLatest(path string) (*Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, &ParseError{
			Path: path,
			Msg:  "failed to open changelog",
			Err:  err,
		}
	}
	defer file.Close()

	headerRegex := regexp.MustCompile(`^#\s*([0-9]+(?:\.[0-9]+){1,2}(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?)\s*-\s*(.+)$`)

	scanner := bufio.NewScanner(file)
	var entry Entry
	collecting := false
	var bulletLines []string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "#") {
			matches := headerRegex.FindStringSubmatch(line)
			if matches == nil {
				continue
			}

			if !collecting {
				entry.Version = strings.TrimSpace(matches[1])
				entry.Summary = strings.TrimSpace(matches[2])
				collecting = true
				continue
			}
			break
		}

		if collecting {
			trimmed := strings.TrimSpace(line)
			if after, found := strings.CutPrefix(trimmed, "-"); found {
				bullet := strings.TrimSpace(after)
				if bullet != "" {
					bulletLines = append(bulletLines, "- "+bullet)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, &ParseError{
			Path: path,
			Msg:  "failed while reading changelog",
			Err:  err,
		}
	}

	if !collecting || entry.Summary == "" {
		return nil, &ParseError{
			Path: path,
			Msg:  fmt.Sprintf("unable to parse latest release entry (expected %s)", ExpectedFormat),
		}
	}

	if len(bulletLines) > 0 {
		entry.Description = strings.Join(bulletLines, "\n")
	}

	return &entry, nil
}
