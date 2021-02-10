package crawler_test

import (
	"context"
	"github.com/google/uuid"
	gc "gopkg.in/check.v1"
	"linkrus/crawler"
	"linkrus/crawler/privnet"
	"linkrus/linkgraph/graph"
	memgraph "linkrus/linkgraph/store/memory"
	"linkrus/textindexer/index"
	memidx "linkrus/textindexer/store/memory"
	"net/http"
	"net/http/httptest"
	"sort"
	"time"
)

var (
	_ = gc.Suite(new(CrawlerIntegrationTestSuite))

	serverRes = `
<html>
	<head>
	  <title>A title</title>
	  <base href="http://google.com/"/>
	</head>
	<body>
	  <a href="./relative">I am a link relative to base</a>
	  <a href="/absolute/path">I am an absolute link</a>
	  <a href="//images/cart.png">I am using the same URL scheme as this page</a>
	  
	  <!-- Link should be added to the index but without creating an edge to it -->
	  <a href="ignore-me" rel="nofollow"/>

	  <!-- The following links should be ignored -->
	  <a href="file:///etc/passwd"></a>
	  <a href="http://169.254.169.254/api/credentials">Link-local address</a>
	</body>
	</html>`
)

type CrawlerIntegrationTestSuite struct{}

func (s *CrawlerIntegrationTestSuite) TestCrawlerPipeline(c *gc.C) {
	linkGraph := memgraph.NewInMemoryGraph()
	searchIndex := mustCreateBleveIndex(c)

	cfg := crawler.Config{
		PrivateNetworkDetector: mustCreatePrivateNetworkDetector(c),
		Graph:                  linkGraph,
		Indexer:                searchIndex,
		URLGetter:              http.DefaultClient,
		FetchWorkers:           5,
	}

	// Start a TLS server and a regular server
	srv1 := mustCreateTestServer(c)
	srv2 := mustCreateTestServer(c)
	defer srv1.Close()
	defer srv2.Close()

	// Import the links
	mustImportLinks(c, linkGraph, []string{
		srv1.URL,
		srv2.URL,
	})

	count, err := crawler.NewCrawler(cfg).Crawl(
		context.Background(),
		mustGetLinkIterator(c, linkGraph),
	)

	c.Assert(err, gc.IsNil)
	c.Assert(count, gc.Equals, 2)

	s.assertGraphLinksMatchList(c, linkGraph, []string{
		srv1.URL,
		srv2.URL,
		"http://google.com/absolute/path",
		"http://google.com/relative",
		"http://google.com/ignore-me",
	})

	s.assertLinksIndexed(c, linkGraph, searchIndex,
		[]string{
			srv1.URL,
			srv2.URL,
		},
		"A title",
		"I am a link relative to base I am an absolute link I am using the same URL scheme as this page Link-local address",
	)
}

func (s *CrawlerIntegrationTestSuite) assertGraphLinksMatchList(c *gc.C, g graph.Graph, exp []string) {
	var got []string
	for it := mustGetLinkIterator(c, g); it.Next(); {
		got = append(got, it.Link().URL)
	}

	sort.Strings(exp)
	sort.Strings(got)
	c.Assert(exp, gc.DeepEquals, got)
}

func (s *CrawlerIntegrationTestSuite) assertLinksIndexed(c *gc.C, g graph.Graph, i index.Indexer, links []string, expTitle, expContent string) {

	var urlToID = make(map[string]uuid.UUID)
	for it := mustGetLinkIterator(c, g); it.Next(); {
		link := it.Link()
		urlToID[link.URL] = link.ID
	}

	zeroTime := time.Time{}
	for _, link := range links {
		id, exists := urlToID[link]
		c.Assert(exists, gc.Equals, true, gc.Commentf("link %q was not retrieved", link))

		doc, err := i.FindByID(id)
		c.Assert(err, gc.IsNil, gc.Commentf("link %q was not added to the search index", link))

		c.Assert(doc.Title, gc.Equals, expTitle)
		c.Assert(doc.Content, gc.Equals, expContent)
		c.Assert(doc.IndexedAt.After(zeroTime), gc.Equals, true, gc.Commentf("indexed document with zero IndexAt timestamp"))
	}
}

// mustCreateTestServer initialise a local server
// that can listen to.
func mustCreateTestServer(c *gc.C) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log the url
		c.Logf("GET %q", r.URL)

		// set the content type to 'application/xhtml'
		w.Header().Set("Content-Type", "application/xhtml")

		// set the status ok
		w.WriteHeader(http.StatusOK)

		// write the serverRes content to the response
		_, err := w.Write([]byte(serverRes))

		// Check the error and should no error
		c.Assert(err, gc.IsNil)
	}))
}

func mustCreateBleveIndex(c *gc.C) *memidx.InMemoryBleveIndexer {
	idx, err := memidx.NewInMemoryBleveIndexer()
	c.Assert(err, gc.IsNil)
	return idx
}

// mustImportLink upsert the links to the graph.
func mustImportLinks(c *gc.C, g graph.Graph, links []string) {
	for _, l := range links {
		err := g.UpsertLink(&graph.Link{
			URL: l,
		})
		c.Logf("importing %q into the graph", l)
		c.Assert(err, gc.IsNil, gc.Commentf("inserting %q", l))
	}
}

func mustCreatePrivateNetworkDetector(c *gc.C) *privnet.Detector {
	det, err := privnet.NewDetectorFromCIDRs("169.254.0.0/16")
	c.Assert(err, gc.IsNil)
	return det
}

func mustGetLinkIterator(c *gc.C, g graph.Graph) graph.LinkIterator {
	it, err := g.Links(uuid.Nil, uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"), time.Now())
	c.Assert(err, gc.IsNil)
	return it
}
