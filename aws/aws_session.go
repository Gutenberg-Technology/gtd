package aws

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/gpkfr/goretdep/config"

	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
)

type AWSSession struct {
	Client client.ConfigProvider
	Svc    *ecs.ECS
}

func NewAWSSession(region *string, profile *string) (*AWSSession, error) {
	awsSession := &AWSSession{}
	var awsConfig *aws.Config

	if *profile != "" {
		awsConfig = &aws.Config{
			Region:      region,
			Credentials: credentials.NewSharedCredentials("", *profile),
		}
		if _, err := awsConfig.Credentials.Get(); err != nil {
			return nil, err
		}
	} else {
		awsConfig = &aws.Config{
			Region: region,
		}
	}

	awsSession.Client = session.Must(session.NewSession(awsConfig))
	return awsSession, nil
}

func (awsSession *AWSSession) GetServiceTask(services *config.Services, svc *ecs.ECS, isDeploy bool, serviceName ...string) error {
	serviceArray := make([]string, 0, 1)

	if serviceCount := len(serviceName); serviceCount <= 0 {
		for _, s := range services.Services {
			if !isDeploy || !s.IgnoreDeploy {
				serviceArray = append(serviceArray, s.Name)
			}
		}
	} else {
		for _, name := range serviceName {
			for _, s := range services.Services {
				if (!isDeploy || !s.IgnoreDeploy) && strings.EqualFold(name, s.Name) {
					serviceArray = append(serviceArray, s.Name)
				}
			}
		}
	}

	serviceArrayCount := len(serviceArray)
	if serviceArrayCount <= 0 {
		err := fmt.Errorf("Missing services")
		return err
	}

	var input *ecs.DescribeServicesInput

	if serviceArrayCount > 10 {
		i := 0
		n := 10

		for {

			input = &ecs.DescribeServicesInput{
				Cluster:  aws.String(services.ECSCluster),
				Services: aws.StringSlice(serviceArray[i : i+n]),
			}
			err := populateConfigServices(services, svc, input)
			if err != nil {
				return fmt.Errorf("GetServiceTask.populateConfigServices. err:%v", err.Error())
			}

			i = i + n
			serviceArrayCount = serviceArrayCount - n
			if serviceArrayCount < n && serviceArrayCount > 0 {
				n = serviceArrayCount
				serviceArrayCount = 0
			} else {
				break
			}

		}
	} else {
		input = &ecs.DescribeServicesInput{
			Cluster:  aws.String(services.ECSCluster),
			Services: aws.StringSlice(serviceArray),
		}
		err := populateConfigServices(services, svc, input)
		if err != nil {
			return fmt.Errorf("GetServiceTask.populateConfigServices. err:%v", err.Error())
		}
	}

	return nil
}
