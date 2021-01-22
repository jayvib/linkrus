// +build unit_tests all_tests

package pipeline_test

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/suite"
	"linkrus/pipeline"
	"sort"
	"testing"
	"time"
)

func TestStageTest(t *testing.T) {
	suite.Run(t, &StageTestSuite{t: t})
}

type StageTestSuite struct {
	suite.Suite
	t *testing.T
}

func (s *StageTestSuite) TestFIFO() {

	makeStages := func(numRunners int) []pipeline.StageRunner {
		stages := make([]pipeline.StageRunner, 10)
		for i := 0; i < len(stages); i++ {
			stages[i] = pipeline.FIFO(makePassthroughProcessor())
		}
		return stages
	}

	s.Suite.Run("success", func() {
		stages := makeStages(10)

		src := &sourceStub{data: stringPayloads(3)}
		sink := new(sinkStub)

		p := pipeline.New(stages...)
		err := p.Process(context.TODO(), src, sink)
		s.NoError(err)
		s.Equal(src.data, sink.data)
		assertAllProcessed(s.Suite, sink.data)
	})

	s.Suite.Run("it cancel the process through context", func() {
		// TODO
	})
}

func (s *StageTestSuite) TestFixedWorkerPool() {
	numWorkers := 10
	syncCh := make(chan struct{})
	rendezvousCh := make(chan struct{})

	proc := pipeline.ProcessorFunc(func(context.Context, pipeline.Payload) (pipeline.Payload, error) {
		syncCh <- struct{}{}

		// Wait for other workers to reach this point
		<-rendezvousCh
		return nil, nil
	})

	src := &sourceStub{data: stringPayloads(numWorkers)}

	p := pipeline.New(pipeline.FixedWorkerPool(proc, numWorkers))

	// Use to wait the processor to finish
	doneCh := make(chan struct{})

	go func() {
		err := p.Process(context.TODO(), src, nil)
		s.NoError(err)
		close(doneCh)
	}()

	// Wait for all workers to reach sync point. This means
	// that each input from the source is currently handled
	// by a worker in parallel.
	for i := 0; i < numWorkers; i++ {
		select {
		case <-syncCh:
		case <-time.After(time.Second * 10):
			s.Suite.T().Fatalf("timed out waiting for worker %d to reach sync point", i)
		}
	}

	close(rendezvousCh)
	select {
	case <-doneCh:
	case <-time.After(10 * time.Second):
		s.Suite.T().Fatal("timed out waiting for pipeline to complete")
	}

}

func (s *StageTestSuite) TestDynamicWorkerPool() {
	numWorker := 5
	syncCh := make(chan struct{}, numWorker)
	rendezvousCh := make(chan struct{})

	proc := pipeline.ProcessorFunc(func(ctx context.Context, payload pipeline.Payload) (pipeline.Payload, error) {
		syncCh <- struct{}{}
		<-rendezvousCh
		return nil, nil
	})

	src := &sourceStub{data: stringPayloads(numWorker * 2)}

	p := pipeline.New(pipeline.DynamicWorkerPool(proc, numWorker))

	doneCh := make(chan struct{})
	go func() {
		err := p.Process(context.TODO(), src, nil)
		s.Suite.NoError(err)
		close(doneCh)
	}()

	for i := 0; i < numWorker; i++ {
		select {
		case <-syncCh:
		case <-time.After(10 * time.Second):
			s.Require().Failf("timed out", "timed out waiting for the worker %d to reach sync point", i)
		}
	}

	close(rendezvousCh)
	select {
	case <-doneCh:
	case <-time.After(10 * time.Second):
		s.Require().Fail("timed out waiting for pipeline to complete")
	}
}

func (s *StageTestSuite) TestBroadcast() {

	numProc := 3
	procs := make([]pipeline.Processor, numProc)
	for i := 0; i < numProc; i++ {
		procs[i] = makeMutatingProcessor(i) // Processor that will write "%val_%index
	}

	src := &sourceStub{data: stringPayloads(1)}
	sink := new(sinkStub)

	p := pipeline.New(pipeline.Broadcast(procs...))
	err := p.Process(context.TODO(), src, sink)
	s.Require().NoError(err)

	expData := []pipeline.Payload{
		&stringPayload{val: "0_0", processed: true},
		&stringPayload{val: "0_1", processed: true},
		&stringPayload{val: "0_2", processed: true},
	}

	assertAllProcessed(s.Suite, src.data)

	sort.Slice(expData, func(i, j int) bool {
		return expData[i].(*stringPayload).val < expData[j].(*stringPayload).val
	})

	sort.Slice(sink.data, func(i, j int) bool {
		return sink.data[i].(*stringPayload).val < sink.data[j].(*stringPayload).val
	})

	s.Equal(expData, sink.data)
}

func makeMutatingProcessor(index int) pipeline.Processor {
	return pipeline.ProcessorFunc(func(ctx context.Context, payload pipeline.Payload) (pipeline.Payload, error) {
		sp := payload.(*stringPayload)
		sp.val = fmt.Sprintf("%s_%d", sp.val, index)
		return payload, nil
	})
}

func makePassthroughProcessor() pipeline.Processor {
	return pipeline.ProcessorFunc(
		func(ctx context.Context, payload pipeline.Payload) (pipeline.Payload, error) {
			return payload, nil
		},
	)
}
