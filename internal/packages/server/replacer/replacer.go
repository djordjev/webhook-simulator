package replacer

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type Replacer interface {
	Replace(str string) (any, error)
}

type stringReplacer struct {
	body   map[string]any
	header http.Header
}

func (s stringReplacer) Replace(str string) (any, error) {
	if !strings.HasPrefix(str, "${{") || !strings.HasSuffix(str, "}}") {
		return str, nil
	}

	noSuffix, _ := strings.CutSuffix(str, "}}")
	value, _ := strings.CutPrefix(noSuffix, "${{")

	//re, err := regexp.Compile(`\$\{\{([^}]*)}}`)
	//if err != nil {
	//	return "", err
	//}
	//
	//res := re.ReplaceAllStringFunc(str, func(match string) string {
	//	log.Println(str, match)
	//
	//	return match
	//})

	//log.Println(res)

	var returnValue any
	var err error

	if strings.HasPrefix(value, "body") {
		noBody, found := strings.CutPrefix(value, "body.")
		if !found {
			return "", errors.New("unable to find path" + str)
		}
		returnValue, err = s.getFromBody(noBody)
	} else if strings.HasPrefix(value, "header") {
		noHeader, found := strings.CutPrefix(value, "header.")
		if !found {
			return "", errors.New("unable to find path" + str)
		}
		returnValue, err = s.getFromHeader(noHeader)
	}

	if err != nil {
		return "", err
	}

	return returnValue, nil
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
