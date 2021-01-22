package pipeline

import "context"

// Payload represents an object that can clone and
// MarkAsProcessed(). This will be the payload that
// will be process by the pipeline
type Payload interface {
	Clone() Payload
	MarkAsProcessed()
}

// Processor is an object that implements the following method.
//
// Context will be use to cancel underlying process. Payload
// will be the payload that will be process by this processor.
// It is a good practice to never mutate the payload parameter.
type Processor interface {
	Process(ctx context.Context, payload Payload) (Payload, error)
}

// ProcessorFunc is an adapter for the Processor interface
type ProcessorFunc func(ctx context.Context, payload Payload) (Payload, error)

// Process is the method for implementing the Processor interface
func (p ProcessorFunc) Process(ctx context.Context, payload Payload) (Payload, error) {
	return p(ctx, payload)
}

// StageParams implementation is a wrapper
// of IO channel payload.
type StageParams interface {
	StageIndex() int
	Input() <-chan Payload
	Output() chan<- Payload
	Error() chan<- error
}

// StageRunner implementation incapsulates of
// running the underlying pipeline.
type StageRunner interface {
	Run(ctx context.Context, params StageParams)
}

// Source implementation provides a valid
// payload to the pipeline.
type Source interface {
	Next(ctx context.Context) bool
	Payload() Payload
	Error() error
}

// Sink drain the output of the pipeline.
type Sink interface {
	Consume(ctx context.Context, payload Payload) error
}
