package crawler

import (
	"context"
	gc "gopkg.in/check.v1"
)

// Register this suite to the global register
var _ = gc.Suite(new(ContentExtractorTestSuite))

type ContentExtractorTestSuite struct{}

func (s *ContentExtractorTestSuite) TestContentExtractor(c *gc.C) {
	content := `
	<div>
		Some<span> content</span> rock &amp; roll
	</div>
	<button>Search</button>
`

	p := new(crawlerPayload)
	_, err := p.RawContent.WriteString(content)
	c.Assert(err, gc.IsNil)

	ret, err := newTextExtractor().Process(context.TODO(), p)
	c.Assert(err, gc.IsNil)
	c.Assert(ret, gc.DeepEquals, p)

	c.Assert(p.Title, gc.Equals, "")
	c.Assert(p.TextContent, gc.Equals, "Some content rock & roll Search")
}
