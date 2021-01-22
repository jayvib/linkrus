package pipeline

import (
	"context"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/xerrors"
	"sync"
)

// New returns a pipeline given on the number of
// stages provided.
func New(stages ...StageRunner) *Pipeline {
	return &Pipeline{stages: stages}
}

// Pipeline executes the a pipeline on a given
// stages.
type Pipeline struct {
	stages []StageRunner
}

// Process run the pipeline and takes the payload from the source and
// send the output to the sink.
func (p *Pipeline) Process(ctx context.Context, source Source, sink Sink) error {
	var wg sync.WaitGroup
	pCtx, ctxCancelFn := context.WithCancel(ctx)

	// +1 to allocate wiring for source
	stageCh := make([]chan Payload, len(p.stages)+1)
	// +2 for source/sink
	errCh := make(chan error, len(p.stages)+2)

	// Fill the stages with a payload channel
	for i := 0; i < len(stageCh); i++ {
		stageCh[i] = make(chan Payload)
	}

	for i := 0; i < len(p.stages); i++ {
		wg.Add(1)
		// Spawn a goroutine as pipeline stage
		go func(stageIndex int) {
			defer func() {
				// Signal the downstream pipeline that
				// there's no more data to be send
				close(stageCh[stageIndex+1])
				wg.Done()
			}()

			p.stages[stageIndex].Run(pCtx, &workerParams{
				stage: stageIndex,
				inCh:  stageCh[stageIndex],
				outCh: stageCh[stageIndex+1],
				errCh: errCh,
			})
		}(i)
	}

	// So all the pipeline are already waiting for the
	// incoming data

	// wait for source and sink goroutines
	wg.Add(2)

	// Paylod Source
	go func() {
		defer func() {
			// Signal the downstream pipeline
			// that there's no more data to process
			close(stageCh[0])
			wg.Done()
		}()

		// Payload source is already sending data to the
		// downstream pipeline
		sourceWorker(pCtx, source, stageCh[0], errCh)
	}()

	// Output sink
	go func() {
		defer func() {
			wg.Done()
		}()
		sinkWorker(pCtx, sink, stageCh[len(stageCh)-1], errCh)
	}()

	go func() {
		// Close the error channel once all workers exit.
		wg.Wait()
		close(errCh)
		ctxCancelFn()
	}()

	// Collect any emitted errors and wrap then in a multi-error
	var err error
	for pErr := range errCh {
		err = multierror.Append(err, pErr)
		ctxCancelFn()
	}

	return err
}

func sourceWorker(ctx context.Context, source Source, outCh chan Payload, errCh chan<- error) {
	for source.Next(ctx) {
		payload := source.Payload()

		select {
		case <-ctx.Done():
			return
		case outCh <- payload:
		}
	}

	if err := source.Error(); err != nil {
		wrappedErr := xerrors.Errorf("pipeline source: %w", err)
		maybeEmitError(wrappedErr, errCh)
	}
}

func sinkWorker(ctx context.Context, sink Sink, inCh <-chan Payload, errChan chan<- error) {
	// TODO
	for {
		select {
		case payload, ok := <-inCh:
			if !ok {
				return
			}

			if err := sink.Consume(ctx, payload); err != nil {
				wrappedErr := xerrors.Errorf("pipeline sink: %w", err)
				maybeEmitError(wrappedErr, errChan)
				return
			}
			payload.MarkAsProcessed()
		case <-ctx.Done():
			return
		}
	}
}

func maybeEmitError(err error, errCh chan<- error) {
	select {
	case errCh <- err:
	default:
	}
}

type workerParams struct {
	stage int
	inCh  <-chan Payload
	outCh chan<- Payload
	errCh chan<- error
}

func (w *workerParams) StageIndex() int {
	return w.stage
}

func (w *workerParams) Input() <-chan Payload {
	return w.inCh
}

func (w *workerParams) Output() chan<- Payload {
	return w.outCh
}

func (w *workerParams) Error() chan<- error {
	return w.errCh
}
