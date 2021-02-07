package crawler

import (
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	gc "gopkg.in/check.v1"
	testifymock "linkrus/crawler/mockery/mocks"
	"linkrus/crawler/mocks"
	"linkrus/linkgraph/graph"
	"testing"
	"time"
)

var _ = gc.Suite(new(GraphUpdaterTestSuite))

type GraphUpdaterTestSuite struct {
	graph *mocks.MockGraph
}

func (g *GraphUpdaterTestSuite) TestGraphUpdater(c *gc.C) {
	logrus.SetLevel(logrus.TraceLevel)
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	g.graph = mocks.NewMockGraph(ctrl)

	payload := &crawlerPayload{
		LinkID: uuid.New(),
		URL:    "http://example.com",
		NoFollowLinks: []string{
			"http://forum.com",
		},
		Links: []string{
			"http://example.com/foo",
			"http://example.com/bar",
		},
	}

	exp := g.graph.EXPECT()

	exp.UpsertLink(linkMatcher{id: payload.LinkID, url: payload.URL, notBefore: time.Now()}).Return(nil)

	id0, id1, id2 := uuid.New(), uuid.New(), uuid.New()
	exp.UpsertLink(linkMatcher{url: "http://forum.com", notBefore: time.Time{}}).DoAndReturn(setLinkID(id0))
	exp.UpsertLink(linkMatcher{url: "http://example.com/foo", notBefore: time.Time{}}).DoAndReturn(setLinkID(id1))
	exp.UpsertLink(linkMatcher{url: "http://example.com/bar", notBefore: time.Time{}}).DoAndReturn(setLinkID(id2))

	// We then expect two edges to be created from the origin link to the
	// two links we just created.
	exp.UpsertEdge(edgeMatcher{src: payload.LinkID, dst: id1}).Return(nil)
	exp.UpsertEdge(edgeMatcher{src: payload.LinkID, dst: id2}).Return(nil)

	exp.RemoveStaleEdges(payload.LinkID, gomock.Any()).Return(nil)

	got := g.updateGraph(c, payload)
	c.Assert(got, gc.Not(gc.IsNil))
}

func (g *GraphUpdaterTestSuite) updateGraph(c *gc.C, p *crawlerPayload) *crawlerPayload {
	out, err := newGraphUpdater(g.graph).Process(context.TODO(), p)
	c.Assert(err, gc.IsNil)
	if out == nil {
		return nil
	}

	// Check if the same type
	c.Assert(out, gc.FitsTypeOf, p)
	return out.(*crawlerPayload)
}

type linkMatcher struct {
	id        uuid.UUID
	url       string
	notBefore time.Time
}

func (l linkMatcher) Matches(x interface{}) bool {
	// Check if the input is a link
	link := x.(*graph.Link)
	logrus.Debug(link.ID, link.URL)
	return l.id == link.ID &&
		l.url == link.URL &&
		!link.RetrievedAt.Before(l.notBefore)
}

func (l linkMatcher) String() string {
	return fmt.Sprintf("has ID=%q, URL=%q and LastAccessed not before %v", l.id, l.url, l.notBefore)
}

func setLinkID(id uuid.UUID) func(*graph.Link) error {
	return func(l *graph.Link) error {
		l.ID = id
		return nil
	}
}

type edgeMatcher struct {
	src uuid.UUID
	dst uuid.UUID
}

func (em edgeMatcher) Matches(x interface{}) bool {
	edge := x.(*graph.Edge)
	return em.src == edge.Src && em.dst == edge.Dst
}

func (em edgeMatcher) String() string {
	return fmt.Sprintf("has Src=%q and Dst=%q", em.src, em.dst)
}

func TestGraphUpdater(t *testing.T) {
	t.SkipNow()
	suite.Run(t, new(GraphUpdaterTestifyMockSuite))
}

type GraphUpdaterTestifyMockSuite struct {
	suite.Suite
}

func (g *GraphUpdaterTestifyMockSuite) Test() {
	payload := &crawlerPayload{
		LinkID: uuid.New(),
		URL:    "http://example.com",
		NoFollowLinks: []string{
			"http://forum.com",
		},
		Links: []string{
			"http://example.com/foo",
			"http://example.com/bar",
		},
	}

	graphMock := new(testifymock.Graph)
	defer graphMock.AssertExpectations(g.Suite.T())
	graphMock.On(
		"UpsertLink",
		mock.MatchedBy(
			newLinkMatcherFunc(
				payload.LinkID,
				payload.URL,
				time.Now(),
			),
		),
	).Return(nil)

	id0, id1, id2 := uuid.New(), uuid.New(), uuid.New()
	graphMock.On(
		"UpsertLink",
		mock.MatchedBy(
			newLinkMatcherFunc(
				uuid.Nil,
				"http://forum.com",
				time.Time{},
			),
		),
	).Return()
	_, _, _ = id0, id1, id2
	out, err := newGraphUpdater(graphMock).Process(context.TODO(), payload)
	g.Assert().NoError(err)

	// Check if the same type
	g.Assert().IsType(payload, out)

}

func newLinkMatcherFunc(id uuid.UUID, url string, notBefore time.Time) func(interface{}) bool {
	return linkMatcher{
		id,
		url,
		notBefore,
	}.Matches
}
