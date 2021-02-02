package crawler

import (
	"context"
	"github.com/golang/mock/gomock"
	gc "gopkg.in/check.v1"
	"linkrus/crawler/mocks"
	"net/url"
	"sort"
)

var _ = gc.Suite(new(ResolveURLTestSuite))
var _ = gc.Suite(new(LinkExtractorTestSuite))

type LinkExtractorTestSuite struct {
	pnd *mocks.MockPrivateNetworkDetector
}

func (l *LinkExtractorTestSuite) TestLinkExtractor(c *gc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	// Setup mock for the PrivateNetworkDetector
	l.pnd = mocks.NewMockPrivateNetworkDetector(ctrl)
	expect := l.pnd.EXPECT()
	expect.IsPrivate("example.com").Return(false, nil).Times(2)
	expect.IsPrivate("foo.com").Return(false, nil).Times(2)

	content := `
<html>
<body>
<a href="https://example.com"/>
<a href="//foo.com"></a>
<a href="/absolute/link"></a>

<!-- the following link should be included in the no follow link list -->
<a href="./local" rel="nofollow"></a>

<!-- duplicates, even with fragments should be skipped -->
<a href="https://example.com#important"/>
<a href="//foo.com"></a>
<a href="/absolute/link#some-anchor"></a>

</body>
</html>
`

	l.assertExtractedLinks(c, "http://test.com", content,
		[]string{
			"https://example.com",
			"http://foo.com",
			"http://test.com/absolute/link",
		}, []string{
			"http://test.com/local",
		})
}

func (l *LinkExtractorTestSuite) assertExtractedLinks(c *gc.C, url, content string, expLinks, expNoFollowLinks []string) {
	p := &crawlerPayload{URL: url}
	// write the content to the body
	_, err := p.RawContent.WriteString(content)
	c.Assert(err, gc.IsNil)

	// New link extractor
	le := newLinkExtractor(l.pnd)
	ret, err := le.Process(context.TODO(), p)
	c.Assert(err, gc.IsNil)
	c.Assert(ret, gc.DeepEquals, p) // Since link extractor mutates the crawler payload

	sort.Strings(expLinks)
	sort.Strings(expNoFollowLinks)

	c.Assert(p.Links, gc.DeepEquals, expLinks)
	c.Assert(p.NoFollowLinks, gc.DeepEquals, expNoFollowLinks)
}

type ResolveURLTestSuite struct{}

func (s *ResolveURLTestSuite) TestResolveAbsoluteURL(c *gc.C) {
	assertResolvedURL(c,
		"/bar/baz",
		"http://example.com/foo/",
		"http://example.com/bar/baz",
	)
}

func (s *ResolveURLTestSuite) TestResolveRelativeURL(c *gc.C) {
	assertResolvedURL(c,
		"bar/baz",
		"http://example.com/foo/",
		"http://example.com/foo/bar/baz",
	)

	assertResolvedURL(c,
		"./bar/baz",
		"http://example.com/foo/secret/",
		"http://example.com/foo/secret/bar/baz",
	)

	assertResolvedURL(c,
		"./bar/baz",
		// Lack of trailing forward slash means we should treat the
		// "secret" as a file and the path is relative to its parent path
		// will be the base.
		"http://example.com/foo/secret",
		"http://example.com/foo/bar/baz",
	)

	assertResolvedURL(c,
		"../../bar/baz",
		"http://example.com/foo/secret",
		"http://example.com/bar/baz",
	)

	assertResolvedURL(c,
		"//www.somewhere.com/foo",
		"http://example.com/bar/secret/",
		"http://www.somewhere.com/foo",
	)
}

func assertResolvedURL(c *gc.C, target, base, want string) {
	// Parse the base url
	baseUrl, err := url.Parse(base)
	c.Assert(err, gc.IsNil)

	// resolve the target url to base
	got := resolveURL(baseUrl, target)
	c.Assert(got, gc.NotNil)

	gotURL := got.String()

	// assert the test expectation
	c.Assert(gotURL, gc.Equals, gotURL)
}
