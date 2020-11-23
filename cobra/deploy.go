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

	// just do a deploy without image replacement
	if strings.EqualFold(newContainerImage, newContainerTag) && !forceDeploy {
		fmt.Println("(💣) Not sure you want to do this. Confirm with `--force`")
		fmt.Println("Quitting Now, bye (🐷)")
		os.Exit(0)
	}

	cmd.AWSSession.GetServices(&cmd.Services, &cmd.Repositories, &cmd.ChildTasks, cmd.GTenv, true, cmd.SelectedServices...)

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Service", "Current Revision", "New Revision", "Current Image", "Desired Image", "status", "Running count"})

	for _, aService := range cmd.Services.Services {
		if aService.TaskDefinition != nil {

			//very Specific to Deploy
			// Check if we use the current image definition
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
					//Add missing colon
					newContainerTag = fmt.Sprintf(":%s", newContainerTag)
				}
			}

			//update tasks
			if forceDeploy || !strings.EqualFold(*aService.TaskDefinition.ContainerDefinitions[0].Image, fmt.Sprintf("%s%s", newContainerImage, newContainerTag)) {
				currentImage = *aService.TaskDefinition.ContainerDefinitions[0].Image
				var isEnvFile bool = false
				//Need to read the environment File

				input := &ecs.RegisterTaskDefinitionInput{
					ContainerDefinitions: aService.TaskDefinition.ContainerDefinitions,
					Family:               aService.TaskDefinition.Family,
					// TaskRoleArn:          aService.TaskDefinition.TaskRoleArn,
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
					// si les valeurs sont différrente / Mis à jour à partir du fichier de config
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
						value := label.Value
						labels[label.Key] = &value
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

				//Update Service
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

				//Ok we have updated service
				//But do we need to publish a ECR, or push Image with another name ?
				if !strings.EqualFold("", aService.UpdateECR) {
					publishRegistry(cmd, t, &aService)
				}

				if aService.UpdateChildTask {
					updateChildTasks(cmd, t, &aService)
				}
			} else {
				// Skipping Update since Current and new Image are identical
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

	// t.SetAllowedColumnLengths([]int{10, -1, 10, 10, 10, 10})
	switch cmd.TableStyle {
	case "light":
		t.SetStyle(table.StyleLight)
	case "color":
		t.SetStyle(table.StyleColoredDark)
	}
	if t.Length() > cmd.ShowTableIndexAbove {
		t.SetAutoIndex(true)
	}
	//t.SetAlign([]text.Align{text.AlignLeft, text.AlignCenter, text.AlignCenter, text.AlignCenter, text.AlignCenter, text.AlignCenter, text.AlignCenter})
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, Align: text.AlignLeft},
		{Number: 2, Align: text.AlignCenter},
		{Number: 3, Align: text.AlignCenter},
		{Number: 4, Align: text.AlignCenter, WidthMax: 30},
		{Number: 5, Align: text.AlignCenter, WidthMax: 30},
		{Number: 6, Align: text.AlignCenter},
		{Number: 7, Align: text.AlignCenter},
	})
	t.Render()
}

func updateChildTasks(cmd *Command, tab interface{}, aService *config.Service) {
	var statusChildTask, currentImage string
	goretPic := "🐺"

	for _, t := range cmd.ChildTasks.ChildTasks {

		if strings.EqualFold(t.ParentService, aService.Name) && !t.IgnoreDeploy {
			taskDefinition, err := cmd.AWSSession.GetCurrentTaskDefinition(cmd.AWSSession.Svc, t.Name)
			if err != nil {
				log.Fatal("error while getting child task definition")
			}

			currentImage = *taskDefinition.TaskDefinition.ContainerDefinitions[0].Image
			taskDefinition.TaskDefinition.ContainerDefinitions[0].Image = aws.String(fmt.Sprintf("%s%s", newContainerImage, newContainerTag))

			_, err = cmd.AWSSession.Svc.RegisterTaskDefinition(&ecs.RegisterTaskDefinitionInput{
				Family:                  taskDefinition.TaskDefinition.Family,
				ContainerDefinitions:    taskDefinition.TaskDefinition.ContainerDefinitions,
				TaskRoleArn:             taskDefinition.TaskDefinition.TaskRoleArn,
				ExecutionRoleArn:        taskDefinition.TaskDefinition.ExecutionRoleArn,
				Memory:                  taskDefinition.TaskDefinition.Memory,
				NetworkMode:             taskDefinition.TaskDefinition.NetworkMode,
				RequiresCompatibilities: taskDefinition.TaskDefinition.RequiresCompatibilities,
				Cpu:                     taskDefinition.TaskDefinition.Cpu,
				Volumes:                 taskDefinition.TaskDefinition.Volumes,
			})

			if err != nil {
				statusChildTask = fmt.Sprintf("Error on %s: %v", t.Name, err)
			} else {
				statusChildTask = fmt.Sprintf("%s for %s Updated", t.Name, t.ParentService)
				goretPic = "🐷"
			}
		} else {
			statusChildTask = "Ignored"
			goretPic = "💤"
		}
		tab.(table.Writer).AppendRow([]interface{}{
			fmt.Sprintf(" ↳ %s", t.Name),
			"-",
			statusChildTask,
			currentImage,
			fmt.Sprintf("same as %s", aService.Name),
			goretPic,
			"-"})
	}
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

				//if ok := len(strings.Split(r.RepositoryName, ":")); ok > 1 {
				repositoryParts := strings.SplitN(r.RepositoryName, ":", 2)
				if len(repositoryParts) == 2 {
					RepositoryNameOnly = repositoryParts[0]
					RepositoryTag = repositoryParts[1]
				} else {
					RepositoryNameOnly = repositoryParts[0]
				}

				//RepositoryTag = fmt.Sprintf(":%s", strings.Split(r.RepositoryName, ":")[1])
				//}
				if !strings.EqualFold("", RepositoryTag) {
					FullURISeparator = ":"
				} else {
					FullURISeparator = ":"
					RepositoryTag = "latest"
				}

				fullURI := fmt.Sprintf("%s%s%s", *RepositoryUri.RepositoryUri, FullURISeparator, RepositoryTag)

				gtddocker.TagLocalDockerImageFrom(*aService.TaskDefinition.ContainerDefinitions[0].Image, fullURI)
				statusChildRegistry = "Tagged Locally (Only)"

				//then push to ecr
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
