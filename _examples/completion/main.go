package main

import (
	"context"
	"fmt"
	"os"

	"github.com/echocat/cli4g"
	"github.com/echocat/cli4g/codec"
)

type options struct {
	Context   string `flag:"context|cluster" short:"c" enum:"development|production" env:"CLI4G_SHIP_CONTEXT" default:"development" description:"Cluster context."`
	Namespace string `flag:"namespace|ns" short:"n" type:"namespace" default:"default" description:"Namespace to use."`
}

type deployOptions struct {
	Service  string `argument:"service" type:"service" required:"true" description:"Service to deploy."`
	Replicas uint32 `flag:"replicas" short:"r" default:"1" description:"Desired replica count."`
	Wait     bool   `flag:"wait" description:"Wait for the planned rollout."`
}

func (opts deployOptions) Validate(context.Context) error {
	if opts.Replicas == 0 {
		return cli.NewValidationError("replicas must be greater than zero")
	}
	return nil
}

func main() {
	cli.NewApplication[options]("ship", "Plan service deployments to a cluster.").
		WithCompletion(true).
		WithCodec(codec.WithCompletionHandler(codec.NewString(), completeNamespaces), "namespace").
		WithCodec(codec.WithCompletionHandler(codec.NewString(), completeServices), "service").
		WithCommand(cli.NewCommand[deployOptions]("deploy", "Plan a service deployment.").
			WithHandler(planDeployment),
		).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func planDeployment(ctx context.Context, opts deployOptions) error {
	application, found := cli.ApplicationFromContext[options](ctx)
	if !found {
		return fmt.Errorf("application options are unavailable")
	}
	if !serviceExists(application.Context, application.Namespace, opts.Service) {
		return cli.NewValidationError("service is unavailable in the selected namespace")
	}
	return cli.Printf(ctx, "Plan: %s -> %s/%s (replicas=%d, wait=%t)\n",
		opts.Service, application.Context, application.Namespace, opts.Replicas, opts.Wait)
}
