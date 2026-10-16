package cli

import "github.com/echocat/cli4g/codec"

func newConfiguredArgument[V, R any](decoder codec.Codec[V, R], name string, configure ...func(*ArgumentRequirement[V, R])) *ArgumentRequirement[V, R] {
	return configureArgument(NewArgument(decoder, name), configure...)
}
func newConfiguredArgumentFactory[V, R any](factory func() (codec.Codec[V, R], error), name string, configure ...func(*ArgumentRequirement[V, R])) *ArgumentRequirement[V, R] {
	return configureArgument(NewArgument[V, R](nil, name).WithCodecFactory(factory), configure...)
}
func configureArgument[V, R any](argument *ArgumentRequirement[V, R], configure ...func(*ArgumentRequirement[V, R])) *ArgumentRequirement[V, R] {
	for _, callback := range configure {
		callback(argument)
	}
	return argument
}
