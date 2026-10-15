package codec

import "context"

// Cumulative aggregates resource-free requirements before one Resolve call.
type Cumulative[R any] interface {
	IsCumulative() bool
	ParseRequirementInto(context.Context, Usage, R, ...string) (R, error)
}

func IsCumulative[V, R any](source Codec[V, R]) bool {
	trait, ok := source.(Cumulative[R])
	return ok && trait.IsCumulative()
}
