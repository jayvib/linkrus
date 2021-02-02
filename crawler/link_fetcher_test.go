package crawler

import (
	"context"
	"github.com/golang/mock/gomock"
	gc "gopkg.in/check.v1"
	"io/ioutil"
	"linkrus/crawler/mocks"
	"net/http"
	"strings"
)

var _ = gc.Suite(new(LinkFetcherTestSuite))

type LinkFetcherTestSuite struct {
	urlGetter       *mocks.MockURLGetter
	privNetDetector *mocks.MockPrivateNetworkDetector
}

func (s *LinkFetcherTestSuite) SetUpTest(c *gc.C) {}

func (s *LinkFetcherTestSuite) TestLinkFetcherWithExcludedExtension(c *gc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()
	s.urlGetter = mocks.NewMockURLGetter(ctrl)
	s.privNetDetector = mocks.NewMockPrivateNetworkDetector(ctrl)

	p := s.fetchLink(c, "http://example.com/foo.png")
	c.Assert(p, gc.IsNil)
}

func (s *LinkFetcherTestSuite) TestLinkFetcher(c *gc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()
	s.urlGetter = mocks.NewMockURLGetter(ctrl)
	s.privNetDetector = mocks.NewMockPrivateNetworkDetector(ctrl)

	s.privNetDetector.EXPECT().IsPrivate("example.com").Return(false, nil)
	s.urlGetter.EXPECT().Get("http://example.com/index.html").Return(
		makeResponse(200, "hello", "application/xhtml"),
		nil,
	)

	p := s.fetchLink(c, "http://example.com/index.html")
	c.Assert(p, gc.NotNil)
	c.Assert(p.RawContent.String(), gc.Equals, "hello")
}

func (s *LinkFetcherTestSuite) fetchLink(c *gc.C, url string) *crawlerPayload {
	p := &crawlerPayload{URL: url}
	out, err := newLinkFetcher(s.urlGetter, s.privNetDetector).Process(context.TODO(), p)
	c.Assert(err, gc.IsNil)

	if out != nil {
		c.Assert(out, gc.FitsTypeOf, p)
		return out.(*crawlerPayload)
	}
	return nil
}

func makeResponse(status int, body, contentType string) *http.Response {
	resp := new(http.Response)
	resp.Body = ioutil.NopCloser(strings.NewReader(body))
	resp.StatusCode = status
	if contentType != "" {
		resp.Header = make(http.Header)
		resp.Header.Set("Content-Type", contentType)
	}
	return resp
}
