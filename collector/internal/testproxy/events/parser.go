package events

import (
	"errors"
	"strings"
)

type Event struct {
	Name        string
	Start       string
	End         string
	Annotations map[string]string
}

func Parse(line string) (*Event, error) {
	var event Event

	const eventPrefix = "@Event "
	if !strings.HasPrefix(line, eventPrefix) {
		return nil, errors.New("line does not start with @Event")
	}
	line = line[len(eventPrefix):]
	startMillis, line := parseMillis(line)
	event.Start = startMillis
	if len(startMillis) == 0 {
		return nil, errors.New("invalid start time")
	}
	endMillis, line := parseMillis(line)
	event.End = endMillis
	if len(endMillis) == 0 {
		return nil, errors.New("invalid end time")
	}
	name, line := endsWith(isLower, isSpace, line)
	event.Name = name
	if len(name) == 0 {
		return nil, errors.New("invalid name")
	}
	event.Annotations = map[string]string{}
	var annotationKey string
	var annotationValue string
	for len(line) > 0 {
		annotationKey, line = endsWith(isLower, isEqual, line)
		if len(annotationKey) == 0 {
			return nil, errors.New("invalid annotation key")
		}
		annotationValue, line = surroundedBy(isAnnotationValue, isQuote, line)
		if len(annotationValue) == 0 {
			return nil, errors.New("invalid annotation value")
		}
		event.Annotations[annotationKey] = annotationValue
		if len(line) > 0 && isSpace([]rune(line)[0]) {
			line = line[1:]
		}
	}
	return &event, nil
}

func isAnnotationValue(r rune) bool {
	return !isQuote(r)
}

func parseMillis(line string) (millis string, rest string) {
	return endsWith(isNumber, isSpace, line)
}

func parseToken(isToken func(rune) bool, line string) (token string, rest string) {
	for _, r := range line {
		if isToken(r) {
			token += string(r)
		} else {
			break
		}
	}
	return token, line[len(token):]
}

func surroundedBy(isToken func(rune) bool, isSeparator func(rune) bool, line string) (string, string) {
	if len(line) == 0 {
		return "", line
	}
	runes := []rune(line)
	if !isSeparator(runes[0]) {
		return "", line
	}
	return endsWith(isToken, isSeparator, string(runes[1:]))
}

func endsWith(isToken func(rune) bool, isSeparator func(rune) bool, line string) (string, string) {
	token, rest := parseToken(isToken, line)
	if len(rest) == 0 {
		return "", line
	}
	runes := []rune(rest)
	if !isSeparator(runes[0]) {
		return "", line
	}
	return token, string(runes[1:])
}

func isQuote(r rune) bool {
	return r == '"'
}

func isEqual(r rune) bool {
	return r == '='
}

func isLower(r rune) bool {
	return 'a' <= r && r <= 'z'
}

func isNumber(r rune) bool {
	return '0' <= r && r <= '9'
}

func isSpace(r rune) bool {
	return r == ' '
}
