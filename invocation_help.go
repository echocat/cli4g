package cli

func (this Invocation[O]) HelpRequested() bool {
	return this.lifecycle != nil && this.lifecycle.helpRequested
}

// HelpHandler returns the application's captured handler, including its global
// fallback. It is usable directly for parse-error responses after cleanup.
func (this Invocation[O]) HelpHandler() HelpHandler {
	if this.application == nil {
		return nil
	}
	return this.application.helpHandler
}

func (this *Application[O]) newHelpInvocation(commands []CommandLike) *Invocation[O] {
	invocation := this.newMetadataInvocation(commands)
	invocation.lifecycle.ready = true
	invocation.lifecycle.helpRequested = true
	return invocation
}

func (this *Application[O]) newMetadataInvocation(commands []CommandLike) *Invocation[O] {
	frames := make([]invocationFrame, len(commands)+1)
	frames[0].container = this
	for i, command := range commands {
		frames[i+1].container = command
	}
	return &Invocation[O]{
		application: this,
		frames:      frames,
		lifecycle: &invocationLifecycle{
			cleanup: func() error { return nil },
		},
	}
}
