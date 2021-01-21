package pipeline

import "context"

type Payload interface {
	Clone() Payload
	MarkAsProcessed()
}

type Processor interface {
	Process(ctx context.Context, payload Payload) (Payload, error)
}

type ProcessorFunc func(ctx context.Context, payload Payload) (Payload, error)

func (p ProcessorFunc) Process(ctx context.Context, payload Payload) (Payload, error) {
	return p(ctx, payload)
}

type StageParams interface {
	StageIndex() int
	Input() <-chan Payload
	Output() chan<- Payload
	Error() chan<- error
}

type StageRunner interface {
	Run(ctx context.Context, params StageParams)
}

type Source interface {
	Next(ctx context.Context) bool
	Payload() Payload
	Error() error
}

type Sink interface {
	Consume(ctx context.Context, payload Payload) error
}
