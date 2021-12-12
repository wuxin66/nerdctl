/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/nerdctl/pkg/dnsutil/hostsstore"
	"github.com/containerd/nerdctl/pkg/idutil/containerwalker"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	// hostsDirBasename is the base name of /var/lib/nerdctl/<ADDRHASH>/etchosts
	hostsDirBasename = "etchosts"
	// metaJSON is stored as /var/lib/nerdctl/<ADDRHASH>/etchosts/<NS>/<ID>/meta.json
	metaJSON = "meta.json"
)

var dataStore string
var ns string

func newRenameCommand() *cobra.Command {
	var renameCommand = &cobra.Command{
		Use:               "rename [flags] CONTAINER ",
		Args:              cobra.MinimumNArgs(2),
		Short:             "Rename a container",
		RunE:              renameAction,
		ValidArgsFunction: renameShellComplete,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}
	return renameCommand
}

func renameAction(cmd *cobra.Command, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("rename requires exactly 2 arguments")
	}
	newContainerName := args[1]
	client, ctx, cancel, err := newClient(cmd)
	if err != nil {
		return err
	}
	defer cancel()
	dataStore, err = getDataStore(cmd)
	if err != nil {
		return err
	}
	ns, err = cmd.Flags().GetString("namespace")
	if err != nil {
		return err
	}

	walker := &containerwalker.ContainerWalker{
		Client: client,
		OnFound: func(ctx context.Context, found containerwalker.Found) error {
			if err := renameContainer(ctx, client, found.Container.ID(), newContainerName); err != nil {
				return err
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\n", found.Req)
			return err
		},
	}
	req := args[0]
	n, err := walker.Walk(ctx, req)
	if err != nil {
		return err
	} else if n == 0 {
		return fmt.Errorf("no such container %s", req)
	}
	return nil
}

func renameContainer(ctx context.Context, client *containerd.Client, id string, newName string) error {
	container, err := client.LoadContainer(ctx, id)
	if err != nil {
		return err
	}
	labels, err := container.Labels(ctx)
	if err != nil {
		return err
	}
	oldName, ok := labels["nerdctl/name"]
	if ok {
		if oldName == newName {
			logrus.Errorf("Renaming a container with the same name as its current name")
		}
	}
	metaPath := getMetaPath(dataStore, ns, id)
	if err != nil {
		return err
	}
	if metaPath != "" {
		var metaData = hostsstore.Meta{}
		metaData, err = readMeta(metaPath)
		if err != nil {
			return err
		}
		metaData.Name = newName
		writeMeta(metaPath, metaData)
	}
	labels["nerdctl/name"] = newName
	_, err = container.SetLabels(ctx, labels)
	if err != nil {
		return err
	}
	return nil
}

func renameShellComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	statusFilterFn := func(st containerd.ProcessStatus) bool {
		return st != containerd.Running && st != containerd.Unknown
	}
	return shellCompleteContainerNames(cmd, statusFilterFn)
}

func getMetaPath(dataStore, ns, id string) string {
	if dataStore == "" || ns == "" || id == "" {
		panic(errdefs.ErrInvalidArgument)
	}
	return filepath.Join(dataStore, hostsDirBasename, ns, id, metaJSON)
}

func readMeta(filename string) (hostsstore.Meta, error) {
	var meta = hostsstore.Meta{}
	metaData, err := ioutil.ReadFile(filename)
	if err != nil {
		return meta, err
	}
	err = json.Unmarshal(metaData, &meta)
	if err != nil {
		return meta, err
	}
	return meta, nil
}

func writeMeta(filename string, meta hostsstore.Meta) error {
	metaB, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(filename, metaB, 0644); err != nil {
		return err
	}
	return nil
}
