package crawler

import (
	"context"
	"io"
	"linkrus/pipeline"
	"net/url"
	"strings"
)

var _ pipeline.Processor = (*linkFetcher)(nil)

func newLinkFetcher(urlGetter URLGetter, netDetector PrivateNetworkDetector) *linkFetcher {
	return &linkFetcher{
		urlGetter:   urlGetter,
		netDetector: netDetector,
	}
}

type linkFetcher struct {
	urlGetter   URLGetter
	netDetector PrivateNetworkDetector
}

func (l *linkFetcher) Process(ctx context.Context, p pipeline.Payload) (pipeline.Payload, error) {
	payload := p.(*crawlerPayload)

	// Check if the link is valid to process
	if exclusionRegex.MatchString(payload.URL) {
		return nil, nil
	}

	// Check if private domain
	if isPrivate, err := l.isPrivate(payload.URL); err != nil || isPrivate {
		return nil, nil
	}

	// Do a HTTP GET request
	res, err := l.urlGetter.Get(payload.URL)
	if err != nil {
		return nil, nil
	}

	// Copy the content to the RawContent
	_, err = io.Copy(&payload.RawContent, res.Body)
	_ = res.Body.Close()
	if err != nil {
		return nil, nil
	}

	// Check the status code
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, nil
	}

	// Skip the non-html response
	if contentType := res.Header.Get("Content-Type"); !strings.Contains(contentType, "html") {
		return nil, nil
	}

	return p, nil
}

func (l *linkFetcher) isPrivate(u string) (bool, error) {
	parsedUrl, err := url.Parse(u)
	if err != nil {
		return false, err
	}

	return l.netDetector.IsPrivate(parsedUrl.Hostname())
}
