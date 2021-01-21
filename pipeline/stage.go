package pipeline

import (
	"context"
	"golang.org/x/xerrors"
	"sync"
)

func FIFO(proc Processor) StageRunner {
	return &fifo{proc: proc}
}

var _ StageRunner = (*fifo)(nil)

type fifo struct {
	proc Processor
}

func (f fifo) Run(ctx context.Context, params StageParams) {

	// Implementation Steps
	// Step 1: Get the Payload from the params
	// Step 2: Process, if there's an error then forward it to error channel
	// Step 3: If payload out is empty then skip it an mark as processed
	// Step 4: Forward the output to the output channel payload
	// Note: Always use the ctx

	for {
		select {
		case <-ctx.Done():
			return
		case payloadIn, ok := <-params.Input():
			if !ok {
				// The upstream stage has no more
				// data to send
				return
			}

			// Process the data
			payloadOut, err := f.proc.Process(ctx, payloadIn)
			if err != nil {
				wrappedErr := xerrors.Errorf(
					"pipeline stage %d: %w",
					params.StageIndex(),
					err,
				)
				maybeEmitError(wrappedErr, params.Error())
				return
			}

			if payloadOut == nil {
				payloadIn.MarkAsProcessed()
				continue
			}

			// Successfully processing the payload
			select {
			// or-done pattern
			case params.Output() <- payloadOut:
			case <-ctx.Done():
				return
			}
		}
	}
}

func FixedWorkerPool(proc ProcessorFunc, numWorkers int) StageRunner {
	if numWorkers <= 0 {
		panic("FixedWorkerPool: numWorkers must be > 0")
	}

	// Initialize the fifox
	fifos := make([]StageRunner, numWorkers)
	for i := 0; i < numWorkers; i++ {
		fifos[i] = FIFO(proc)
	}

	return &fixedWorkerPool{fifos: fifos}
}

// This is use to fan-out a stage
type fixedWorkerPool struct {
	fifos []StageRunner
}

func (p *fixedWorkerPool) Run(ctx context.Context, params StageParams) {

	// Spin up each worker in the pool and wait for then to exit
	var wg sync.WaitGroup

	for i := 0; i < len(p.fifos); i++ {
		wg.Add(1)
		go func(indexRunner int) {
			defer wg.Done()
			p.fifos[indexRunner].Run(ctx, params)
		}(i)
	}

	// Wait for the goroutines to exit
	wg.Wait()
}

func DynamicWorkerPool(proc ProcessorFunc, numWorkers int) StageRunner {
	if numWorkers <= 0 {
		panic("DynamicWorkerPool: maxWorkers must be > 0")
	}

	// Initialize the semaphore
	tokenPool := make(chan struct{}, numWorkers)
	for i := 0; i < numWorkers; i++ {
		tokenPool <- struct{}{}
	}

	return &dynamicWorkerPool{
		proc: proc,
		tokenPool: tokenPool,
	}
}

type dynamicWorkerPool struct {
	proc      Processor
	tokenPool chan struct{} // Semaphore
}

func (d *dynamicWorkerPool) Run(ctx context.Context, params StageParams) {
stop:
	for {
		select {
		case <-ctx.Done():
			break stop
		case payloadIn, ok := <-params.Input():
			if !ok {
				break stop
			}

			// Check if there's available token
			var token struct{}
			select {
			case token = <-d.tokenPool:
			case <-ctx.Done():
				break stop
			}

			// Fire a goroutine
			go func(payloadIn Payload, token struct{}) {
				defer func() {
					// Reuse the token after use to
					// signal this dynamic worker pool
					// that there's an available slot
					d.tokenPool <- token
				}()

				// Process
				payloadOut, err := d.proc.Process(ctx, payloadIn)
				if err != nil {
					wrappedErr := xerrors.Errorf("pipeline stage %d: %w", params.StageIndex(), err)
					maybeEmitError(wrappedErr, params.Error())
					return
				}

				// Successfully process the payload
				if payloadOut == nil {
					// There's something happen in the upstream
					// but no error. By this means...just mark
					// the payload input as processed then exit
					payloadIn.MarkAsProcessed()
					return
				}

				// Send the payload out to downstream
				select {
				case <-ctx.Done():
					return
				case params.Output() <- payloadOut:
				}
			}(payloadIn, token)
		}
	}

	// Wait for all workers to exit by trying to empty the token pool
	for i := 0; i < cap(d.tokenPool); i++ {
		<-d.tokenPool
	}
}

func Broadcast(procs ...Processor) StageRunner {

	if len(procs) == 0 {
		panic("Broadcast: at least one processor must be specified")
	}

	fifos := make([]StageRunner, len(procs))
	for i, p := range procs {
		fifos[i] = FIFO(p)
	}

	return &broadcast{fifos: fifos}
}

type broadcast struct {
	fifos []StageRunner
}

func (b *broadcast) Run(ctx context.Context, params StageParams) {
	var (
		wg sync.WaitGroup
		inCh = make([]chan Payload, len(b.fifos))
	)

	// Start each FIFO in a go-routine. Each FIFO gets its own dedicated
	// input channel and the shared output channel passed to Run.
	for i := 0; i < len(b.fifos); i++ {
		wg.Add(1)
		inCh[i] = make(chan Payload)
		go func(fifoIndex int) {
			defer wg.Done()
			fifoParams := &workerParams{
				stage: params.StageIndex(),
				inCh: inCh[fifoIndex],
				outCh: params.Output(),
				errCh: params.Error(),
			}
			b.fifos[fifoIndex].Run(ctx, fifoParams)
		}(i)
	}

done:
	for {
		select {
		case <-ctx.Done():
			break done
		case payloadIn, ok := <-params.Input():
			if !ok {
				break done
			}

			// Send the payloadIn to the input channels
			// for each runners.
			for i := len(b.fifos)-1; i >= 0; i-- {
				// As each FIFO might modify the payload, to
				// avoid data races we need to make a copy of
				// the payload for all FIFOs except the first.
				var fifoPayload = payloadIn

				if i != 0 {
					fifoPayload = payloadIn.Clone()
				}

				select {
				case <-ctx.Done():
					return
				case inCh[i] <- fifoPayload:
				}
			}
		}
	}

	// Close input channels and wait FIFOs to exit
	for _, ch := range inCh {
		close(ch)
	}

	wg.Wait()
}