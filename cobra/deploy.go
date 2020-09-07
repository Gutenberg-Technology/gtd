package cobra

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/gpkfr/goretdep/config"
	"github.com/gpkfr/goretdep/gtddocker"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/spf13/cobra"
)

var (
	newContainerImage   string
	newContainerTag     string
	forceDeploy         bool
	environmentFilePath string
)

func NewDeployCommand(cmd *Command) {
	cobraCmd := &cobra.Command{
		Use:   "deploy",
		Short: "update service to use new task revision",

		Run: func(cobraCmd *cobra.Command, args []string) {
			deployServices(cmd)
		},
		PreRun: func(cobraCmd *cobra.Command, args []string) {
			cmd.CheckEnv()
			cmd.GetAWSSession()
		},
	}

	cobraCmd.Flags().StringVarP(&cmd.GTenv, "env", "e", "", "Environment to deploy")
	cobraCmd.Flags().StringSliceVarP(&cmd.SelectedServices, "service", "s", []string{}, "Service(s) to deploy. Separated by comma")
	cobraCmd.Flags().StringVarP(&newContainerImage, "container-image", "c", "", "Container Image to deploy")
	cobraCmd.Flags().StringVarP(&newContainerTag, "tag", "t", "", "tag of Image to deploy")
	cobraCmd.Flags().BoolVar(&forceDeploy, "force", false, "Force new deployement")
	cobraCmd.Flags().StringVar(&environmentFilePath, "config", "", "Task's Config file (Environment)")

	cmd.AddCommand(cobraCmd)
}

func deployServices(cmd *Command) {
	var currentImage string
	var newServiceTaskDefinition string = "Unmodified"

	if strings.EqualFold(newContainerImage, newContainerTag) && !forceDeploy {
		fmt.Println("(💣) Not sure you want to do this. Confirm with `--force`")
		fmt.Println("Quitting Now, bye (🐷)")
		os.Exit(0)
	}

	cmd.AWSSession.GetServices(&cmd.Services, &cmd.Repositories, cmd.GTenv, true, cmd.SelectedServices...)

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Service", "Current Revision", "New Revision", "Current Image", "Desired Image", "status", "Running count"})

	for _, aService := range cmd.Services.Services {
		if aService.TaskDefinition != nil {

			if newContainerImage == "" {
				newContainerImage = aService.Registry
				if newContainerTag == "" && forceDeploy {
					newContainerImage = *aService.TaskDefinition.ContainerDefinitions[0].Image
				}
			}

			if newContainerTag != "" {
				if strings.Contains(newContainerImage, ":") {
					log.Fatal(fmt.Errorf("Tags already defined in %s\n", newContainerImage))
				}
				if !strings.Contains(newContainerTag, ":") {
					newContainerTag = fmt.Sprintf(":%s", newContainerTag)
				}
			}

			if forceDeploy || !strings.EqualFold(*aService.TaskDefinition.ContainerDefinitions[0].Image, fmt.Sprintf("%s%s", newContainerImage, newContainerTag)) {
				currentImage = *aService.TaskDefinition.ContainerDefinitions[0].Image
				var isEnvFile bool = false

				input := &ecs.RegisterTaskDefinitionInput{
					ContainerDefinitions: aService.TaskDefinition.ContainerDefinitions,
					Family:               aService.TaskDefinition.Family,
				}

				if aService.TaskDefinition.TaskRoleArn != nil {
					if !strings.EqualFold(aService.TaskRoleArn, *aService.TaskDefinition.TaskRoleArn) {
						input.SetTaskRoleArn(aService.TaskRoleArn)
						isEnvFile = true
					}
				} else {
					if !strings.EqualFold("", aService.TaskRoleArn) {
						input.SetTaskRoleArn(aService.TaskRoleArn)
						isEnvFile = true
					}
				}

				if aService.TaskDefinition.ExecutionRoleArn != nil {
					if !strings.EqualFold(aService.TaskExecutionRoleArn, *aService.TaskDefinition.ExecutionRoleArn) {
						input.SetExecutionRoleArn(aService.TaskExecutionRoleArn)
						isEnvFile = true
					} else {
						input.SetExecutionRoleArn(aService.TaskExecutionRoleArn)
					}

				} else {
					if !strings.EqualFold("", aService.TaskExecutionRoleArn) {
						input.SetExecutionRoleArn(aService.TaskExecutionRoleArn)
						isEnvFile = true
					}
				}

				if !strings.EqualFold("", environmentFilePath) {
					log.Printf("Read Env File %s", environmentFilePath)
					taskEnv, err := config.ReadTaskEnvFile(environmentFilePath)
					if err != nil {
						log.Fatal(fmt.Errorf("Error while Reading %s", environmentFilePath))
					}
					isEnvFile = true

					tasksEnv := make([]*ecs.KeyValuePair, 0)
					secretsEnv := make([]*ecs.Secret, 0)
					for k, v := range taskEnv {
						if strings.HasPrefix(k, "_") {
							k := strings.TrimPrefix(k, "_")
							secretsEnv = append(secretsEnv, &ecs.Secret{
								Name:      aws.String(k),
								ValueFrom: aws.String(v),
							})
						} else {
							tasksEnv = append(tasksEnv, &ecs.KeyValuePair{
								Name:  aws.String(k),
								Value: aws.String(v),
							})
						}
					}
					input.ContainerDefinitions[0].SetEnvironment(tasksEnv)
					input.ContainerDefinitions[0].SetSecrets(secretsEnv)

					if len(secretsEnv) > 0 {
						input.SetExecutionRoleArn(aService.TaskExecutionRoleArn)
					}
				}

				labels := make(map[string]*string)

				if len(aService.Labels) > 0 {
					for _, label := range aService.Labels {
						labels[label.Key] = &label.Value
					}

				}

				if eq := reflect.DeepEqual(input.ContainerDefinitions[0].DockerLabels, labels); !eq {
					input.ContainerDefinitions[0].SetDockerLabels(labels)
					isEnvFile = true
				}

				if isEnvFile || !strings.EqualFold(*aService.TaskDefinition.ContainerDefinitions[0].Image, fmt.Sprintf("%s%s", newContainerImage, newContainerTag)) {
					input.ContainerDefinitions[0].SetImage(fmt.Sprintf("%s%s", newContainerImage, newContainerTag))

					result, err := cmd.AWSSession.Svc.RegisterTaskDefinition(input)
					if err != nil {
						log.Fatal(fmt.Errorf("error while registering task definifition : %s\n%s", *aService.TaskDefinition.Family, err.Error()))
					}
					newServiceTaskDefinition = fmt.Sprintf("%s:%d", *result.TaskDefinition.Family, *result.TaskDefinition.Revision)
				} else {
					newServiceTaskDefinition = fmt.Sprintf("%s:%d", *aService.TaskDefinition.Family, *aService.TaskDefinition.Revision)
				}

				_, err := cmd.AWSSession.UpdateAWSService(cmd.AWSSession.Svc, &aService.Name, &cmd.Services.ECSCluster, &newServiceTaskDefinition, forceDeploy)
				if err != nil {
					log.Println(fmt.Errorf("error while updating service: %s\n %s", aService.Name, err.Error()))
				}

				t.AppendRow([]interface{}{
					aService.Name,
					fmt.Sprintf("%s:%d", *aService.TaskDefinition.Family, *aService.TaskDefinition.Revision),
					newServiceTaskDefinition,
					currentImage,
					fmt.Sprintf("%s%s", newContainerImage, newContainerTag),
					aService.Status,
					aService.RunningCount})

				if !strings.EqualFold("", aService.UpdateECR) {
					publishRegistry(cmd, t, &aService)
				}
			} else {
				t.AppendRow([]interface{}{
					aService.Name,
					fmt.Sprintf("%s:%d", *aService.TaskDefinition.Family, *aService.TaskDefinition.Revision),
					newServiceTaskDefinition,
					*aService.TaskDefinition.ContainerDefinitions[0].Image,
					fmt.Sprintf("%s%s", newContainerImage, newContainerTag),
					aService.Status,
					aService.RunningCount})
			}
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
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, Align: text.AlignLeft},
		{Number: 2, Align: text.AlignCenter},
		{Number: 3, Align: text.AlignCenter},
		{Number: 4, Align: text.AlignCenter},
		{Number: 5, Align: text.AlignCenter},
		{Number: 6, Align: text.AlignCenter},
		{Number: 7, Align: text.AlignCenter},
	})
	t.Render()
}

func publishRegistry(cmd *Command, t interface{}, aService *config.Service) {
	statusChildRegistry := "-"
	goretPic := "🐺"
	var RepositoryNameOnly, RepositoryTag, FullURISeparator string

	fmt.Printf("Service name (Source): %s\nImage: %s\n", aService.Name, *aService.TaskDefinition.ContainerDefinitions[0].Image)

	for _, r := range cmd.Repositories.Repositories {

		if strings.EqualFold(r.Name, aService.UpdateECR) && !r.IgnoreDeploy {

			if gtddocker.PullFromPrivateRegistry(cmd.DockerHubAuthConfig, *aService.TaskDefinition.ContainerDefinitions[0].Image) {
				RepositoryUri := cmd.AWSSession.DescribeRepository(r.RepositoryName)

				repositoryParts := strings.SplitN(r.RepositoryName, ":", 2)
				if len(repositoryParts) == 2 {
					RepositoryNameOnly = repositoryParts[0]
					RepositoryTag = repositoryParts[1]
				} else {
					RepositoryNameOnly = repositoryParts[0]
				}

				if !strings.EqualFold("", RepositoryTag) {
					FullURISeparator = ":"
				} else {
					FullURISeparator = ":"
					RepositoryTag = "latest"
				}

				fullURI := fmt.Sprintf("%s%s%s", *RepositoryUri.RepositoryUri, FullURISeparator, RepositoryTag)

				gtddocker.TagLocalDockerImageFrom(*aService.TaskDefinition.ContainerDefinitions[0].Image, fullURI)
				statusChildRegistry = "Tagged Locally (Only)"

				if cmd.AWSSession.PushToECR(RepositoryNameOnly, RepositoryTag, fullURI) {
					statusChildRegistry = fmt.Sprintf("Pushed on %s", fullURI)
					goretPic = "🐷"
				}

			}
		} else {
			statusChildRegistry = "Ignored"
			goretPic = "💤"
		}
		t.(table.Writer).AppendRow([]interface{}{
			fmt.Sprintf(" ↳ %s", aService.UpdateECR),
			"-",
			statusChildRegistry,
			"-",
			"-",
			goretPic,
			"-"})
	}
}
