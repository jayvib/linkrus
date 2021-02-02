package crawler

import (
	"context"
	"linkrus/pipeline"
	"net/url"
	"regexp"
	"strings"
)

var (
	exclusionRegex = regexp.MustCompile(`(?i)\.(?:jpg|jpeg|png|gif|ico|css|js)$`)
	baseHrefRegex  = regexp.MustCompile(`(?i)<base.*?href\s*?=\s*?"(.*?)\s*?"`)
	findLinkRegex  = regexp.MustCompile(`(?i)<a.*?href\s*?=\s*?"\s*?(.*?)\s*?".*?>`)
	nofollowRegex  = regexp.MustCompile(`(?i)rel\s*?=\s*?"?nofollow"?`)
)

func newLinkExtractor(p PrivateNetworkDetector) *linkExtractor {
	return &linkExtractor{}
}

type linkExtractor struct {
}

func (l *linkExtractor) Process(ctx context.Context, p pipeline.Payload) (pipeline.Payload, error) {
	return p, nil
}

// resolveURL expands target into an absolute URL using the following rules:
// 1. targets starting with '//' are treated as abolute URLs that inherit the
//    protocol from relTo.
// 2. targets starting with '/' are absolute URLs that are appended to the host from relTo.
// 3. all other targets are assumed tto be relative to relTo
//
// If the target URL cannot be parsed, a nil URL will be returned.
func resolveURL(relTo *url.URL, target string) *url.URL {

	tLen := len(target)
	if tLen == 0 {
		return nil
	}

	if strings.HasPrefix(target, "//") {
		target = relTo.Scheme + ":" + target
	}

	if targetURL, err := url.Parse(target); err == nil {
		return relTo.ResolveReference(targetURL)
	}

	return nil
}
