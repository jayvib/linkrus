package crawler

import (
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	gc "gopkg.in/check.v1"
	"linkrus/crawler/mocks"
	"linkrus/textindexer/index"
	"time"
)

var _ = gc.Suite(new(TextIndexerTestSuite))

type TextIndexerTestSuite struct {
	indexer *mocks.MockIndexer
}

func (s *TextIndexerTestSuite) TestTextIndexer(c *gc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	s.indexer = mocks.NewMockIndexer(ctrl)

	payload := &crawlerPayload{
		LinkID:      uuid.New(),
		URL:         "http://example.com",
		Title:       "some title",
		TextContent: "Lorem ipsum dolor",
	}

	exp := s.indexer.EXPECT()

	exp.Index(docMatcher{
		linkID:    payload.LinkID,
		url:       payload.URL,
		title:     payload.Title,
		content:   payload.TextContent,
		notBefore: time.Now(),
	}).Return(nil)

	out, err := newTextIndexer(s.indexer).Process(context.TODO(), payload)
	c.Assert(err, gc.IsNil)

	c.Assert(out, gc.NotNil)
	c.Assert(out, gc.FitsTypeOf, payload)
}

type docMatcher struct {
	linkID    uuid.UUID
	url       string
	title     string
	content   string
	notBefore time.Time
}

func (d docMatcher) Matches(x interface{}) bool {
	doc := x.(*index.Document)
	return doc.LinkID == d.linkID &&
		doc.URL == d.url &&
		doc.Title == d.title &&
		doc.Content == doc.Content &&
		!doc.IndexedAt.Before(d.notBefore)
}

func (d docMatcher) String() string {
	return fmt.Sprintf(
		"has LinkID=%q, URL=%q, Title=%q, Content=%q and IndexedAt not before %v",
		d.linkID, d.url, d.title, d.content, d.notBefore,
	)
}
