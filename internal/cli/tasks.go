package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"text/tabwriter"
	"time"

	"github.com/ccheshirecat/viper/internal/types"
	"github.com/ccheshirecat/viper/pkg/client"
	"github.com/spf13/cobra"
)

func taskCmd() *cobra.Command {
	task := &cobra.Command{
		Use:   "tasks",
		Short: "Manage automation tasks",
		Long:  "Submit, monitor, and retrieve results from browser automation tasks.",
	}

	task.AddCommand(taskSubmitCmd())
	task.AddCommand(taskLogsCmd())
	task.AddCommand(taskScreenshotsCmd())
	task.AddCommand(taskStatusCmd())

	return task
}

func taskSubmitCmd() *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "submit [vm] [task.json]",
		Short: "Submit a task to a VM agent",
		Long:  "Submit a browser automation task defined in a JSON file to the specified VM.",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			vmName := args[0]
			taskFile := args[1]

			data, err := ioutil.ReadFile(taskFile)
			checkError(err)

			var task types.Task
			err = json.Unmarshal(data, &task)
			checkError(err)

			task.VMID = vmName
			if timeout > 0 {
				task.Timeout = timeout
			}

			agentClient, err := client.NewAgentClient(vmName)
			checkError(err)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, err := agentClient.SubmitTask(ctx, task)
			checkError(err)

			fmt.Printf("Task submitted successfully\n")
			fmt.Printf("Task ID: %s\n", result.TaskID)
			fmt.Printf("Status: %s\n", result.Status)
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", 60*time.Second, "Task execution timeout")

	return cmd
}

func taskLogsCmd() *cobra.Command {
	var follow bool

	cmd := &cobra.Command{
		Use:   "logs [vm] [task-id]",
		Short: "Fetch task logs",
		Long:  "Retrieve execution logs for a specific task.",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			vmName := args[0]
			taskID := args[1]

			agentClient, err := client.NewAgentClient(vmName)
			checkError(err)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			logs, err := agentClient.GetTaskLogs(ctx, taskID)
			checkError(err)

			fmt.Print(logs)
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output (stream)")

	return cmd
}

func taskScreenshotsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "screenshots [vm] [task-id]",
		Short: "List screenshots for a task",
		Long:  "Display available screenshots captured during task execution.",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			vmName := args[0]
			taskID := args[1]

			agentClient, err := client.NewAgentClient(vmName)
			checkError(err)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			screenshots, err := agentClient.GetTaskScreenshots(ctx, taskID)
			checkError(err)

			if len(screenshots) == 0 {
				fmt.Println("No screenshots found for this task")
				return
			}

			fmt.Printf("Screenshots for task %s:\n", taskID)
			for i, screenshot := range screenshots {
				fmt.Printf("  %d. %s\n", i+1, screenshot)
			}
		},
	}

	return cmd
}

func taskStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [vm] [task-id]",
		Short: "Get task status",
		Long:  "Display detailed status information for a specific task.",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			vmName := args[0]
			taskID := args[1]

			agentClient, err := client.NewAgentClient(vmName)
			checkError(err)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			task, err := agentClient.GetTaskStatus(ctx, taskID)
			checkError(err)

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "Task ID:\t%s\n", task.ID)
			fmt.Fprintf(w, "VM ID:\t%s\n", task.VMID)
			fmt.Fprintf(w, "Status:\t%s\n", task.Status)
			fmt.Fprintf(w, "URL:\t%s\n", task.URL)
			fmt.Fprintf(w, "Created:\t%s\n", task.Created.Format("2006-01-02 15:04:05"))

			if task.Started != nil {
				fmt.Fprintf(w, "Started:\t%s\n", task.Started.Format("2006-01-02 15:04:05"))
			}

			if task.Completed != nil {
				fmt.Fprintf(w, "Completed:\t%s\n", task.Completed.Format("2006-01-02 15:04:05"))
				if task.Started != nil {
					duration := task.Completed.Sub(*task.Started)
					fmt.Fprintf(w, "Duration:\t%s\n", duration)
				}
			}

			if task.Error != "" {
				fmt.Fprintf(w, "Error:\t%s\n", task.Error)
			}

			w.Flush()
		},
	}

	return cmd
}