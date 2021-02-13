// +build unit_tests all_tests

package pipeline_test

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/suite"
	"linkrus/pipeline"
	"testing"
	"github.com/bradfitz/gomemcache/memcache"
)

func TestStageTestSuite(t *testing.T) {
	memcache.New()
	suite.Run(t, &PipelineTestSuite{t: t})
}

type PipelineTestSuite struct {
	suite.Suite
	t *testing.T
}

func (s *PipelineTestSuite) TestDataFlow() {
	// Create 10 instances of stage runner
	stages := make([]pipeline.StageRunner, 10)
	// fill the stages
	for i := 0; i < len(stages); i++ {
		stages[i] = testStage{s: s}
	}

	// Make an instance of the payload source
	src := &sourceStub{data: stringPayloads(3)}
	sink := new(sinkStub)

	p := pipeline.New(stages...)
	err := p.Process(context.TODO(), src, sink)
	s.Suite.NoError(err)
	s.Suite.Equal(src.data, sink.data)
	assertAllProcessed(s.Suite, src.data)
}

type sourceStub struct {
	index int
	data  []pipeline.Payload
	err   error
}

func (s *sourceStub) Next(context.Context) bool {
	if s.err != nil || s.index == len(s.data) {
		return false
	}
	s.index++
	return true
}

func (s *sourceStub) Error() error { return s.err }
func (s *sourceStub) Payload() pipeline.Payload {
	return s.data[s.index-1]
}

type sinkStub struct {
	data []pipeline.Payload
	err  error
}

func (s *sinkStub) Consume(_ context.Context, p pipeline.Payload) error {
	s.data = append(s.data, p)
	return s.err
}

type stringPayload struct {
	processed bool
	val       string
}

func (s *stringPayload) Clone() pipeline.Payload { return &stringPayload{val: s.val} }
func (s *stringPayload) MarkAsProcessed()        { s.processed = true }
func (s *stringPayload) String() string          { return s.val }

func stringPayloads(numValues int) []pipeline.Payload {
	out := make([]pipeline.Payload, numValues)
	for i := 0; i < numValues; i++ {
		out[i] = &stringPayload{val: fmt.Sprint(i)}
	}
	return out
}

func assertAllProcessed(s suite.Suite, data []pipeline.Payload) {
	for i, p := range data {
		payload := p.(*stringPayload)
		s.Equal(true, payload.processed, "not processed %s", i)
	}
}

type testStage struct {
	s           *PipelineTestSuite
	dropPayload bool
	err         error
}

func (t testStage) Run(ctx context.Context, params pipeline.StageParams) {
	for {
		select {
		case <-ctx.Done():
			return
		case p, ok := <-params.Input():
			if !ok {
				return
			}

			if t.err != nil {
				t.s.t.Logf("[stage %d] emit error: %v", params.StageIndex(), t.err)
				params.Error() <- t.err
				return
			}

			select {
			case <-ctx.Done():
				return
			case params.Output() <- p:
			}
		}
	}
}
