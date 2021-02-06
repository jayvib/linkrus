package crawler

import (
	"context"
	"github.com/microcosm-cc/bluemonday"
	"html"
	"linkrus/pipeline"
	"regexp"
	"strings"
	"sync"
)

var (
	titleRegex         = regexp.MustCompile(`(?i)<title.*?>(.*?)</title>`)
	repeatedSpaceRegex = regexp.MustCompile(`\s+`)
)

type textExtractor struct {
	policyPool sync.Pool
}

func newTextExtractor() *textExtractor {
	return &textExtractor{
		policyPool: sync.Pool{
			New: func() interface{} {
				return bluemonday.StrictPolicy()
			},
		},
	}
}

func (te *textExtractor) Process(_ context.Context, p pipeline.Payload) (pipeline.Payload, error) {
	payload := p.(*crawlerPayload)
	// Get the policy sanitizer
	policy := te.policyPool.Get().(*bluemonday.Policy)
	defer func() {
		// Put back the policy sanitizer to the pool
		te.policyPool.Put(policy)
	}()

	sp := new(stringProcessor)

	sanitizeString := func(s string) string {
		sp.set(s)
		defer sp.set("")
		sp.
			// Sanitize the html to readable text
			process(policy.Sanitize).
			// Remove the redundant spaces
			process(replaceAllRepeatedSpace(" ")).
			// Convert the url encoded string to readable string
			process(html.UnescapeString).
			// Trim all the leading and trailing spaces
			process(strings.TrimSpace)
		return sp.get()
	}

	// Match a title via regex.
	if titleMatch := titleRegex.FindStringSubmatch(payload.RawContent.String()); len(titleMatch) == 2 {
		// The first group will be the value of the title
		// Sanitize the matched title
		// Remove all redundant spaces
		// Convert the URL encoded string to readable UTF-8 string
		// Trim leading and trailing spaces

		payload.Title = sanitizeString(titleMatch[1])
	}

	payload.TextContent = sanitizeString(payload.RawContent.String())

	return payload, nil
}

type stringProcessor struct {
	str string
}

func (s *stringProcessor) process(fn func(string) string) *stringProcessor {
	s.str = fn(s.str)
	return s
}

func (s *stringProcessor) set(str string) {
	s.str = str
}

func (s *stringProcessor) get() string {
	return s.str
}

func replaceAllRepeatedSpace(with string) func(string) string {
	return func(str string) string {
		return repeatedSpaceRegex.ReplaceAllString(str, with)
	}
}
