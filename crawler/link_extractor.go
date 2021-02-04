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
	return &linkExtractor{netDetector: p}
}

type linkExtractor struct {
	netDetector PrivateNetworkDetector
}

func (l *linkExtractor) Process(ctx context.Context, p pipeline.Payload) (pipeline.Payload, error) {
	payload := p.(*crawlerPayload)
	relTo, err := url.Parse(payload.URL)
	if err != nil {
		return nil, err
	}

	content := payload.RawContent.String()

	// Check if there's a base html tag..
	if baseMatch := baseHrefRegex.FindStringSubmatch(content); len(baseMatch) == 2 {
		if base := resolveURL(relTo, ensureHasTrailingSlash(baseMatch[1])); base != nil {
			relTo = base
		}
	}

	// Find the unique set of links from the document, resolve then and
	// add them to the payload.
	seenMap := make(map[string]struct{})
	for _, match := range findLinkRegex.FindAllStringSubmatch(content, -1) {
		// Resolve to the base url
		hrefLink := match[1]
		link := resolveURL(relTo, hrefLink) // the first

		// Skip the link that aren't valid
		if !l.retainLink(relTo.Hostname(), link) {
			continue
		}

		// Truncate anchors and drop duplicates
		link.Fragment = ""

		linkStr := link.String()

		// Skip the duplicate link
		if _, seen := seenMap[linkStr]; seen {
			continue
		}

		// Skip the links that aren't html
		if exclusionRegex.MatchString(linkStr) {
			continue
		}

		seenMap[linkStr] = struct{}{}

		// Check if no follow link
		if nofollowRegex.MatchString(match[0]) {
			payload.NoFollowLinks = append(payload.NoFollowLinks, linkStr)
		} else {
			payload.Links = append(payload.Links, linkStr)
		}
	}

	return p, nil
}

func (l *linkExtractor) retainLink(srcHost string, link *url.URL) bool {
	if link == nil {
		return false
	}

	// Skip links that aren't http(s) schemes
	if link.Scheme != "http" && link.Scheme != "https" {
		return false
	}

	// Keep links to the same host
	if link.Hostname() == srcHost {
		return true
	}

	// Skip links that resolve to private networks
	if isPrivate, err := l.netDetector.IsPrivate(link.Host); err != nil || isPrivate {
		return false
	}

	return true
}

func ensureHasTrailingSlash(path string) string {
	if !strings.HasSuffix(path, "/") {
		return path + "/"
	}
	return path
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
