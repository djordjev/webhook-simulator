package replacer

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const VariableRegexp = `\$\{\{([^}]*)}}`
const timeOffset = `(after|before) ([0-9]+) (millisecond|second|minute|hour|day)s?`

var now = time.Now
var newUUID = uuid.New

type Replacer interface {
	Replace(str string) (any, error)
}

type stringReplacer struct {
	body   map[string]any
	header http.Header
}

func (s stringReplacer) Replace(str string) (any, error) {
	re, err := regexp.Compile(VariableRegexp)
	if err != nil {
		return "", err
	}

	matches := re.FindAllString(str, -1)
	if len(matches) == 0 {
		return str, nil
	}

	if len(matches) == 1 {
		return s.doReplacement(matches[0])
	}

	res := re.ReplaceAllStringFunc(str, func(match string) string {
		replaced, replacementError := s.doReplacement(match)
		if replacementError != nil {
			return ""
		}

		return fmt.Sprint(replaced)
	})

	return res, nil
}

func (s stringReplacer) doReplacement(variable string) (any, error) {
	variable = variable[3 : len(variable)-2]
	if strings.HasPrefix(variable, "body.") {
		value, prefixFound := strings.CutPrefix(variable, "body.")
		if !prefixFound {
			return "", errors.New("unable to cut body. from" + variable)
		}

		return s.getFromBody(value)
	}

	if strings.HasPrefix(variable, "header.") {
		value, prefixFound := strings.CutPrefix(variable, "header.")
		if !prefixFound {
			return "", errors.New("unable to cut header. from" + variable)
		}

		return s.getFromHeader(value)
	}

	if variable == "now" {
		return s.getCurrentDate(), nil
	}

	timeOffsetRegex, err := regexp.Compile(timeOffset)
	if err != nil {
		return "", err
	}

	timeOffsetMatches := timeOffsetRegex.FindAllString(variable, -1)
	if len(timeOffsetMatches) > 0 {
		return s.getTimeOffset(variable), nil
	}

	if variable == "uuid" {
		return s.getUUID(), nil
	}

	if strings.HasPrefix(variable, "random") {
		return s.getRandomInt(variable), nil
	}

	if strings.HasPrefix(variable, "digit") {
		return s.getRandomDigit(variable), nil
	}

	if strings.HasPrefix(variable, "letter") {
		return s.getRandomLetter(variable), nil
	}

	return "", nil
}

func (s stringReplacer) getRandomInt(value string) int {
	segments := strings.Split(value, " ")

	minVal := 0
	maxVal := 1000000

	var err error

	if len(segments) == 3 {
		maxVal, err = strconv.Atoi(segments[2])
		if err != nil {
			return 0
		}
	}

	if len(segments) >= 2 {
		minVal, err = strconv.Atoi(segments[1])
		if err != nil {
			return 0
		}
	}

	return rand.Intn(maxVal-minVal) + minVal
}

func (s stringReplacer) getRandomLetter(value string) string {
	var err error
	segments := strings.Split(value, " ")
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	count := 1
	if len(segments) == 2 {
		count, err = strconv.Atoi(segments[1])
		if err != nil {
			return ""
		}
	}

	var buffer bytes.Buffer
	for range count {
		rnd := letterRunes[rand.Intn(len(letterRunes))]
		buffer.WriteRune(rnd)
	}

	return buffer.String()
}

func (s stringReplacer) getRandomDigit(value string) string {
	var err error
	segments := strings.Split(value, " ")

	count := 1
	if len(segments) == 2 {
		count, err = strconv.Atoi(segments[1])
		if err != nil {
			return ""
		}
	}

	var buffer bytes.Buffer
	for i := 0; i < count; i++ {
		random := rand.Intn(10)
		buffer.WriteString(fmt.Sprint(random))
	}

	return buffer.String()
}

func (s stringReplacer) getCurrentDate() string {
	return now().UTC().Format(time.RFC3339)
}

func (s stringReplacer) getUUID() string {
	return newUUID().String()
}

func (s stringReplacer) getTimeOffset(value string) string {
	segments := strings.Split(value, " ")
	measureSegment := segments[2]

	offset, err := strconv.Atoi(segments[1])
	if err != nil {
		return ""
	}

	var measure time.Duration
	if strings.HasPrefix(measureSegment, "millisecond") {
		measure = time.Millisecond
	} else if strings.HasPrefix(measureSegment, "second") {
		measure = time.Second
	} else if strings.HasPrefix(measureSegment, "minute") {
		measure = time.Minute
	} else if strings.HasPrefix(measureSegment, "hour") {
		measure = time.Hour
	} else if strings.HasPrefix(measureSegment, "day") {
		measure = time.Hour * 24
	} else {
		return ""
	}

	if segments[0] == "before" {
		offset *= -1
	}

	return now().Add(time.Duration(offset) * measure).UTC().Format(time.RFC3339)
}

func (s stringReplacer) getFromBody(value string) (any, error) {
	segments := strings.Split(value, ".")

	current := s.body
	length := len(segments)

	for k, v := range segments {
		isLast := k == length-1

		currentVal, found := current[v]

		if !found {
			return "", fmt.Errorf("unable to find segment %s in path %s", v, value)
		}

		if isLast {
			return currentVal, nil
		} else {
			if next, ok := currentVal.(map[string]any); ok {
				current = next
			} else {
				return "", errors.New("next value is not a map" + value)
			}
		}
	}

	return "", errors.New("not found for path" + value)
}

func (s stringReplacer) getFromHeader(value string) (any, error) {
	val := s.header.Get(value)

	if val == "" {
		return "", errors.New("cant find in header " + value)
	}

	return val, nil
}

func NewReplacer(body map[string]any, header http.Header) Replacer {
	return stringReplacer{body: body, header: header}
}
