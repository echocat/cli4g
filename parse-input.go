package cli

// parseInput contains invocation-local tokens, before decoding any values.
type parseInput struct {
	commands      []CommandLike
	flags         []flagInput
	positions     [][]string
	helpRequested bool
}

// appendFlags reserves space for a bounded prefix of adjacent option-looking
// tokens when the first flags arrive. It does not interpret those tokens: the
// scanner still resolves commands, arity and -- exactly once. Positional tails
// do not contribute to the hint, and empty inputs allocate no flag buffer.
func (this *parseInput) appendFlags(remaining []string, flags ...flagInput) {
	const maxInitialFlagCapacity = 128
	if this.flags == nil {
		capacity := len(flags)
		for _, token := range remaining {
			if capacity >= maxInitialFlagCapacity || len(token) < 2 || token == "--" || token[0] != '-' {
				break
			}
			capacity++
		}
		if capacity > len(flags) {
			this.flags = make([]flagInput, 0, capacity)
		}
	}
	this.flags = append(this.flags, flags...)
}
