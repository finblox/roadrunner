package cli

import (
	"fmt"
	"log"
	"net/rpc"
	"os"
	"time"

	tm "github.com/buger/goterm"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spiral/errors"
	"github.com/spiral/roadrunner-plugins/informer"
	"github.com/spiral/roadrunner/v2/tools"
)

var (
	interactive bool
)

const InformerList string = "informer.List"

func init() {
	workersCommand := &cobra.Command{
		Use:   "workers",
		Short: "Show information about active roadrunner workers",
		RunE:  workersHandler,
	}

	workersCommand.Flags().BoolVarP(
		&interactive,
		"interactive",
		"i",
		false,
		"render interactive workers table",
	)

	root.AddCommand(workersCommand)
}

func workersHandler(cmd *cobra.Command, args []string) error {
	const op = errors.Op("workers handler")
	// get RPC client
	client, err := RPCClient()
	if err != nil {
		return err
	}
	defer func() {
		err := client.Close()
		if err != nil {
			log.Printf("error when closing RPCClient: error %v", err)
		}
	}()

	var plugins []string
	// assume user wants to show workers from particular plugin
	if len(args) != 0 {
		plugins = args
	} else {
		err = client.Call(InformerList, true, &plugins)
		if err != nil {
			return errors.E(op, err)
		}
	}

	if !interactive {
		return showWorkers(plugins, client)
	}

	tm.Clear()
	tt := time.NewTicker(time.Second)
	defer tt.Stop()
	for {
		select {
		case <-tt.C:
			tm.MoveCursor(1, 1)
			err := showWorkers(plugins, client)
			if err != nil {
				return errors.E(op, err)
			}
			tm.Flush()
		}
	}
}

func showWorkers(plugins []string, client *rpc.Client) error {
	for _, plugin := range plugins {
		list := &informer.WorkerList{}
		err := client.Call("informer.Workers", plugin, &list)
		if err != nil {
			return err
		}

		// it's a golang :)
		ps := make([]tools.ProcessState, len(list.Workers))
		for i := 0; i < len(list.Workers); i++ {
			ps[i].Created = list.Workers[i].Created
			ps[i].NumJobs = list.Workers[i].NumJobs
			ps[i].MemoryUsage = list.Workers[i].MemoryUsage
			ps[i].Pid = list.Workers[i].Pid
			ps[i].Status = list.Workers[i].Status
		}

		fmt.Printf("Workers of [%s]:\n", color.HiYellowString(plugin))
		tools.WorkerTable(os.Stdout, ps).Render()
	}
	return nil
}
