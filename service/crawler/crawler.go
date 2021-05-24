package crawler

import (
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/juju/clock"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
	"io/ioutil"
	"linkrus/crawler"
	"linkrus/crawler/privnet"
	"linkrus/linkgraph/graph"
	"linkrus/partition"
	"linkrus/textindexer/index"
	"net/http"
	"time"
)

//go:generate mockgen -package mocks -destination mocks/mocks.go . GraphAPI,IndexAPI

type GraphAPI interface {
	UpsertLink(link *graph.Link) error
	UpsertEdge(edge *graph.Edge) error
	RemoveStaleEdges(fromID uuid.UUID, updatedBefore time.Time) error
	Links(fromID, toID uuid.UUID, retrievedBefore time.Time) (graph.LinkIterator, error)
}

type IndexAPI interface {
	Index(doc *index.Document) error
}

type Config struct {
	GraphAPI               GraphAPI
	IndexAPI               IndexAPI
	PrivateNetworkDetector crawler.PrivateNetworkDetector
	URLGetter              crawler.URLGetter
	PartitionDetector      partition.Detector
	Clock                  clock.Clock
	FetchWorkers           int
	UpdateInterval         time.Duration
	ReIndexThreshold       time.Duration
	Logger                 *logrus.Entry
}

func (cfg Config) Clone() Config {
	return cfg
}

func (cfg *Config) validate() error {
	var err error
	if cfg.PrivateNetworkDetector == nil {
		cfg.PrivateNetworkDetector, err = privnet.NewDetector()
	}
	if cfg.URLGetter == nil {
		cfg.URLGetter = http.DefaultClient
	}
	if cfg.GraphAPI == nil {
		err = multierror.Append(err, xerrors.Errorf("graph API has not been provided"))
	}
	if cfg.IndexAPI == nil {
		err = multierror.Append(err, xerrors.Errorf("index API has not been provided"))
	}
	if cfg.PartitionDetector == nil {
		err = multierror.Append(err, xerrors.Errorf("partition detector has not been provided"))
	}
	if cfg.Clock == nil {
		cfg.Clock = clock.WallClock
	}
	if cfg.FetchWorkers <= 0 {
		err = multierror.Append(err, xerrors.Errorf("invalid value for fetch workers"))
	}
	if cfg.ReIndexThreshold == 0 {
		err = multierror.Append(err, xerrors.Errorf("invalid value for re-index threshold"))
	}
	if cfg.Logger == nil {
		cfg.Logger = logrus.NewEntry(&logrus.Logger{Out: ioutil.Discard})
	}
	return err
}

type Service struct {
	cfg     Config
	crawler *crawler.Crawler
}

func NewService(cfg Config) (*Service, error) {
	if err := cfg.validate(); err != nil {
		return nil, xerrors.Errorf("crawler service: config validation failed: %w", err)
	}

	return &Service{
		cfg: cfg,
		crawler: crawler.NewCrawler(crawler.Config{
			PrivateNetworkDetector: cfg.PrivateNetworkDetector,
			URLGetter:              cfg.URLGetter,
			Graph:                  cfg.GraphAPI,
			Indexer:                cfg.IndexAPI,
			FetchWorkers:           cfg.FetchWorkers,
		}),
	}, nil
}

func (svc *Service) Name() string { return "crawler" }
