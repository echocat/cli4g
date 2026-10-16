package cli

// HelpInfo is a fresh metadata view of the selected path. Definition pointers
// refer to immutable compiled snapshots. Mutating the returned lists does not
// change the application. No options, codecs, defaults or handlers are executed.
type HelpInfo struct {
	Application Container
	Target      Container
	Path        []Container
	Commands    []CommandLike
	Flags       []HelpFlagInfo
	Arguments   []HelpArgumentInfo
}

type HelpFlagInfo struct {
	Definition *Flag[any, any]
	Owner      Container
	Inherited  bool
	MinValues  int
	MaxValues  int // -1 means variadic
}

type HelpArgumentInfo struct {
	Definition *Argument[any, any]
	MinValues  int
	MaxValues  int
}

func (this Invocation[O]) HelpInfo() HelpInfo {
	info := HelpInfo{Application: this.application}
	if len(this.frames) == 0 {
		return info
	}
	info.Path = make([]Container, len(this.frames))
	for i := range this.frames {
		info.Path[i] = this.frames[i].container
	}
	info.Target = info.Path[len(info.Path)-1]
	plan := info.Target.base().plan
	if len(this.selected) > 0 {
		info.Commands = append(info.Commands, this.selected[len(this.selected)-1].commands...)
	} else {
		for command := range info.Target.Commands() {
			info.Commands = append(info.Commands, command)
		}
	}
	for i := range info.Path {
		schema := &plan.schemas[i]
		if len(this.selected) > 0 {
			schema = this.selected[i].schema
		}
		for _, flag := range schema.flags {
			info.Flags = append(info.Flags, HelpFlagInfo{
				Definition: flag,
				Owner:      info.Path[i],
				Inherited:  i < len(info.Path)-1,
				MinValues:  flag.arity.min,
				MaxValues:  flag.arity.max,
			})
		}
		if i == len(info.Path)-1 {
			for _, argument := range schema.arguments {
				info.Arguments = append(info.Arguments, HelpArgumentInfo{
					Definition: argument,
					MinValues:  argument.arity.min,
					MaxValues:  argument.arity.max,
				})
			}
		}
	}
	return info
}
