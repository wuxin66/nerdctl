package main

import (
	"github.com/containerd/containerd"
	"github.com/urfave/cli/v2"
)

var restartCommand = &cli.Command{
	Name:         "restart",
	Usage:        "restart one or more running containers",
	ArgsUsage:    "[flags] CONTAINER [CONTAINER, ...]",
	Action:       restartAction,
	BashComplete: restartBashComplete,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "time",
			Aliases: []string{"t"},
			Usage:   "Seconds to wait for stop before killing it",
			Value:   "10",
		},
	},
}

func restartAction(clicontext *cli.Context) error {
	err := stopAction(clicontext)
	if err != nil {
		return err
	}
	err = startAction(clicontext)
	if err != nil {
		return err
	}
	return nil
}

func restartBashComplete(clicontext *cli.Context) {
	coco := parseCompletionContext(clicontext)
	if coco.boring || coco.flagTakesValue {
		defaultBashComplete(clicontext)
		return
	}

	// show non-stopped container names
	statusFilterFn := func(st containerd.ProcessStatus) bool {
		return st != containerd.Stopped && st != containerd.Created && st != containerd.Unknown
	}
	bashCompleteContainerNames(clicontext, statusFilterFn)
	// show non-running container names
	statusFilterFn = func(st containerd.ProcessStatus) bool {
		return st != containerd.Running && st != containerd.Unknown
	}
	bashCompleteContainerNames(clicontext, statusFilterFn)
}
