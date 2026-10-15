package cli

import (
	"context"
	"github.com/echocat/cli4g/codec"
)

func (c grammarString) ParseRequirement(ctx context.Context, usage codec.Usage, args ...string) (string, error) {
	return c.Parse(ctx, usage, args...)
}
func (c grammarString) FormatRequirement(ctx context.Context, value string) ([]string, error) {
	return c.Format(ctx, value)
}
func (c grammarString) DisplayRequirement(ctx context.Context, value string) ([]string, error) {
	return c.Format(ctx, value)
}
func (c grammarString) CloneRequirement(ctx context.Context, value string) (string, error) {
	if c.cloneErr != nil {
		return "", c.cloneErr
	}
	return value, ctx.Err()
}
func (c grammarString) Resolve(ctx context.Context, _ codec.Scope, value string) (string, error) {
	return value, ctx.Err()
}

func (c *constantOneCodec) ParseRequirement(ctx context.Context, usage codec.Usage, args ...string) (int, error) {
	if err := codec.EnsureArgumentsMatching(ctx, usage, c, args...); err != nil {
		return 0, err
	}
	return 1, nil
}
func (c *constantOneCodec) FormatRequirement(ctx context.Context, value int) ([]string, error) {
	return codec.NewInt().FormatRequirement(ctx, value)
}
func (c *constantOneCodec) DisplayRequirement(ctx context.Context, value int) ([]string, error) {
	return c.FormatRequirement(ctx, value)
}
func (c *constantOneCodec) Resolve(ctx context.Context, _ codec.Scope, value int) (int, error) {
	return value, ctx.Err()
}

func (c flagChunkCodec) ParseRequirement(ctx context.Context, usage codec.Usage, args ...string) (flagChunkValues, error) {
	return c.Parse(ctx, usage, args...)
}
func (c flagChunkCodec) FormatRequirement(ctx context.Context, value flagChunkValues) ([]string, error) {
	return c.Format(ctx, value)
}
func (c flagChunkCodec) DisplayRequirement(ctx context.Context, value flagChunkValues) ([]string, error) {
	return c.Format(ctx, value)
}
func (c flagChunkCodec) CloneRequirement(ctx context.Context, value flagChunkValues) (flagChunkValues, error) {
	return c.CloneValue(ctx, value)
}
func (c flagChunkCodec) Resolve(ctx context.Context, _ codec.Scope, value flagChunkValues) (flagChunkValues, error) {
	return value, ctx.Err()
}

func (this custom1Codec) ParseRequirement(ctx context.Context, usage codec.Usage, args ...string) (custom1, error) {
	return this.Parse(ctx, usage, args...)
}
func (this custom1Codec) FormatRequirement(ctx context.Context, value custom1) ([]string, error) {
	return this.Format(ctx, value)
}
func (this custom1Codec) DisplayRequirement(ctx context.Context, value custom1) ([]string, error) {
	return this.Format(ctx, value)
}
func (this custom1Codec) Resolve(ctx context.Context, _ codec.Scope, value custom1) (custom1, error) {
	return value, ctx.Err()
}

func (this customVariadic1Codec) ParseRequirement(ctx context.Context, usage codec.Usage, args ...string) (customVariadic1, error) {
	return this.Parse(ctx, usage, args...)
}
func (this customVariadic1Codec) FormatRequirement(ctx context.Context, value customVariadic1) ([]string, error) {
	return this.Format(ctx, value)
}
func (this customVariadic1Codec) DisplayRequirement(ctx context.Context, value customVariadic1) ([]string, error) {
	return this.Format(ctx, value)
}
func (this customVariadic1Codec) Resolve(ctx context.Context, _ codec.Scope, value customVariadic1) (customVariadic1, error) {
	return value, ctx.Err()
}
