package aws

import (
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/gpkfr/goretdep/config"
)

func (awsSession *AWSSession) GetSVC() {
	awsSession.Svc = ecs.New(awsSession.Client)
}

func (awsSession *AWSSession) GetCurrentTaskDefinition(svc *ecs.ECS, taskARN string) (*ecs.DescribeTaskDefinitionOutput, error) {
	input := &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(taskARN),
	}

	result, err := awsMust(svc.DescribeTaskDefinition(input))
	return result.(*ecs.DescribeTaskDefinitionOutput), err
}

func (awsSession *AWSSession) UpdateAWSService(svc *ecs.ECS, serviceName, serviceCluster, taskDefinition *string, forceDeploy bool) (*ecs.UpdateServiceOutput, error) {
	input := &ecs.UpdateServiceInput{
		Cluster:            serviceCluster,
		Service:            serviceName,
		ForceNewDeployment: aws.Bool(forceDeploy),
		TaskDefinition:     taskDefinition,
	}
	result, err := awsMust(svc.UpdateService(input))
	return result.(*ecs.UpdateServiceOutput), err
}

func (awsSession *AWSSession) GetServices(services *config.Services, repos *config.Repositories, child *config.ChildTasks, env string, isDeploy bool, selectedServices ...string) {
	if err := config.LoadService(services, repos, child, &env); err != nil {
		log.Fatal(err)
	}

	awsSession.GetSVC()

	if len(selectedServices) <= 0 {
		//No filter
		err := awsSession.GetServiceTask(services, awsSession.Svc, isDeploy)
		if err != nil {
			log.Fatal(err)
		}

	} else {
		//filter on Services Selected
		err := awsSession.GetServiceTask(services, awsSession.Svc, isDeploy, selectedServices...)
		if err != nil {
			log.Fatal(err)
		}
	}

	for i, s := range services.Services {
		if !strings.EqualFold("", s.TaskARN) {
			currentTask, err := awsSession.GetCurrentTaskDefinition(awsSession.Svc, s.TaskARN)
			if err != nil {
				log.Println(err)
			}
			services.Services[i].TaskDefinition = currentTask.TaskDefinition
		}
	}
}
