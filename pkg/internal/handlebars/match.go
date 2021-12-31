package handlebars

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/aymerick/raymond"
)

var regexRegistry = map[string]string{}
var regexRegistryCount = 1
var idRegex = regexp.MustCompile(`tesh-match-\d+-\d+`)

func init() {
	raymond.RegisterHelper("match", func(regex string, options *raymond.Options) string {
		return registerRegex(regex)
	})
}

func registerRegex(regex string) string {
	id := fmt.Sprintf("tesh-match-%d-%d", regexRegistryCount, time.Now().UnixNano())
	regexRegistryCount += 1
	regexRegistry[id] = regex
	return id
}

func ExpandRegexes(s string) (string, bool) {
	hasRegex := false
	s = idRegex.ReplaceAllStringFunc(s, func(id string) string {
		if regex, ok := regexRegistry[id]; ok {
			hasRegex = true
			// delete(regexRegistry, id)
			return regex
		} else {
			fmt.Fprintf(os.Stderr, "can't find regex with id: %s", id)
			return id
		}
	})

	return s, hasRegex
}
