package cobra

import (
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func NewStatusCommand(cmd *Command) {

	cobraCmd := &cobra.Command{
		Use:   "status",
		Short: "service's status",

		Run: func(cobraCmd *cobra.Command, args []string) {
			status(cmd)
		},
		PreRun: func(cobraCmd *cobra.Command, args []string) {
			cmd.CheckEnv()
			cmd.GetAWSSession()
		},
	}

	cobraCmd.Flags().StringVarP(&cmd.GTenv, "env", "e", "", "Environment to show")
	cobraCmd.Flags().StringSliceVarP(&cmd.SelectedServices, "service", "s", []string{}, "Service(s) to show. Separated by comma")

	cmd.AddCommand(cobraCmd)
}

func status(cmd *Command) {

	cmd.AWSSession.GetServices(&cmd.Services, &cmd.Repositories, cmd.GTenv, false, cmd.SelectedServices...)

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Service", "Family", "Revision", "Current Image", "status", "Running count"})
	// loop under services
	for _, aService := range cmd.Services.Services {
		if aService.TaskDefinition != nil {
			t.AppendRow([]interface{}{
				aService.Name,
				*aService.TaskDefinition.Family,
				*aService.TaskDefinition.Revision,
				*aService.TaskDefinition.ContainerDefinitions[0].Image,
				aService.Status,
				aService.RunningCount})
		}
	}

	switch cmd.TableStyle {
	case "light":
		t.SetStyle(table.StyleLight)
	case "color":
		t.SetStyle(table.StyleColoredDark)
	}
	if t.Length() > cmd.ShowTableIndexAbove {
		t.SetAutoIndex(true)
	}
	t.Render()
}
