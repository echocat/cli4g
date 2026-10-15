package codec

import (
	"context"
	"time"
)

func (this stringValue) DisplayRequirement(ctx context.Context, value string) ([]string, error) {
	return this.FormatRequirement(ctx, value)
}
func (this duration) DisplayRequirement(ctx context.Context, value time.Duration) ([]string, error) {
	return this.FormatRequirement(ctx, value)
}
func (this boolean) DisplayRequirement(ctx context.Context, value bool) ([]string, error) {
	return this.FormatRequirement(ctx, value)
}
func (this signed[T]) DisplayRequirement(ctx context.Context, value T) ([]string, error) {
	return this.FormatRequirement(ctx, value)
}
func (this unsigned[T]) DisplayRequirement(ctx context.Context, value T) ([]string, error) {
	return this.FormatRequirement(ctx, value)
}
func (this floating[T]) DisplayRequirement(ctx context.Context, value T) ([]string, error) {
	return this.FormatRequirement(ctx, value)
}
func (this complexValue[T]) DisplayRequirement(ctx context.Context, value T) ([]string, error) {
	return this.FormatRequirement(ctx, value)
}
func (this textCodec[T]) DisplayRequirement(ctx context.Context, value string) ([]string, error) {
	return this.FormatRequirement(ctx, value)
}
func (this tokenValue[T]) DisplayRequirement(ctx context.Context, value []string) ([]string, error) {
	return this.FormatRequirement(ctx, value)
}
