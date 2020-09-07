package aws

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/gpkfr/goretdep/config"
)

func awsMust(result interface{}, err error) (interface{}, error) {

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecs.ErrCodeServerException:
				fmt.Println(ecs.ErrCodeServerException, aerr.Error())
			case ecs.ErrCodeClientException:
				fmt.Println(ecs.ErrCodeClientException, aerr.Error())
			case ecs.ErrCodeInvalidParameterException:
				fmt.Println(ecs.ErrCodeInvalidParameterException, aerr.Error())
			case ecs.ErrCodeClusterNotFoundException:
				fmt.Println(ecs.ErrCodeClusterNotFoundException, aerr.Error())
			case ecs.ErrCodeServiceNotFoundException:
				fmt.Println(ecs.ErrCodeServiceNotFoundException, aerr.Error())
			case ecs.ErrCodeServiceNotActiveException:
				fmt.Println(ecs.ErrCodeServiceNotActiveException, aerr.Error())
			case ecs.ErrCodePlatformUnknownException:
				fmt.Println(ecs.ErrCodePlatformUnknownException, aerr.Error())
			case ecs.ErrCodePlatformTaskDefinitionIncompatibilityException:
				fmt.Println(ecs.ErrCodePlatformTaskDefinitionIncompatibilityException, aerr.Error())
			case ecs.ErrCodeAccessDeniedException:
				fmt.Println(ecs.ErrCodeAccessDeniedException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return nil, err
	}
	return result, nil
}

func populateConfigServices(services *config.Services, svc *ecs.ECS, input *ecs.DescribeServicesInput) error {

	result, err := awsMust(svc.DescribeServices(input))
	if err != nil {
		return err
	}
	//finally Populate Services Struct
	if resultCount := len(result.(*ecs.DescribeServicesOutput).Services); resultCount > 0 {
		var found int
		for _, awsService := range result.(*ecs.DescribeServicesOutput).Services {
			for i, gtService := range services.Services {
				if strings.EqualFold(*awsService.ServiceName, gtService.Name) {
					services.Services[i].TaskARN = *awsService.Deployments[0].TaskDefinition
					services.Services[i].Status = *awsService.Status
					services.Services[i].RunningCount = *awsService.RunningCount
					found++
				}
			}
			if found == resultCount {
				break
			}
		}
	}
	return nil

}
