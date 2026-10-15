package cli

import "github.com/echocat/cli4g/codec"

func newConfiguredFlag[V, R any](decoder codec.Codec[V, R], name string, configure ...func(*FlagRequirement[V, R])) *FlagRequirement[V, R] {
	return configureFlag(NewFlag(decoder).WithName(name), configure...)
}
func newConfiguredFlagFactory[V, R any](factory func() (codec.Codec[V, R], error), name string, configure ...func(*FlagRequirement[V, R])) *FlagRequirement[V, R] {
	return configureFlag(NewFlag[V, R](nil).WithName(name).WithCodecFactory(factory), configure...)
}
func configureFlag[V, R any](flag *FlagRequirement[V, R], configure ...func(*FlagRequirement[V, R])) *FlagRequirement[V, R] {
	for _, callback := range configure {
		callback(flag)
	}
	return flag
}
