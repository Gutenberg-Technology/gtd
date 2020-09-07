package aws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
)

func (awsSession *AWSSession) DescribeRepository(repositoryName string) *ecr.Repository {

	repositoryName = strings.Split(repositoryName, ":")[0]
	svc := ecr.New(awsSession.Client)

	input := &ecr.DescribeRepositoriesInput{
		RepositoryNames: aws.StringSlice([]string{repositoryName})}

	result, err := svc.DescribeRepositories(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecr.ErrCodeServerException:
				fmt.Println(ecr.ErrCodeServerException, aerr.Error())
			case ecr.ErrCodeInvalidParameterException:
				fmt.Println(ecr.ErrCodeInvalidParameterException, aerr.Error())
			case ecr.ErrCodeRepositoryNotFoundException:
				fmt.Println(ecr.ErrCodeRepositoryNotFoundException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return nil
	}
	if len(result.Repositories) > 0 {
		return result.Repositories[0]
	}
	return nil
}

func (awsSession *AWSSession) GetDockerAuthStrFromEcr(svc *ecr.ECR, RegistryId *string) *string {

	tokenInput := ecr.GetAuthorizationTokenInput{

		RegistryIds: aws.StringSlice([]string{*RegistryId}),
	}

	tokenOutput, err := awsMust(svc.GetAuthorizationToken(&tokenInput))
	if err != nil {
		log.Fatal(err)
	}
	decodedToken, err := base64.StdEncoding.DecodeString(aws.StringValue(tokenOutput.(*ecr.GetAuthorizationTokenOutput).AuthorizationData[0].AuthorizationToken))
	if err != nil {
		_ = fmt.Errorf("GetDockerAuthStrFromEcr.decodetoken. err:%v", err)
	}

	parts := strings.SplitN(string(decodedToken), ":", 2)

	authConfig := types.AuthConfig{
		Username:      parts[0],
		Password:      parts[1],
		ServerAddress: aws.StringValue(tokenOutput.(*ecr.GetAuthorizationTokenOutput).AuthorizationData[0].ProxyEndpoint),
	}

	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		_ = fmt.Errorf("GetDockerAuthStrFromEcr.encodedJson. err:%v", err)
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	return &authStr

}

func (awsSession *AWSSession) PushToECR(repositoryName, repositoryTag, fullURI string) bool {
	fmt.Println("")
	svc := ecr.New(awsSession.Client)

	if repository := awsSession.DescribeRepository(repositoryName); repository != nil {
		//get AuthStr
		authStr := awsSession.GetDockerAuthStrFromEcr(svc, repository.RegistryId)
		if strings.EqualFold("", *authStr) {
			return false
		}

		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			return false
		}

		if strings.HasPrefix(fullURI, aws.StringValue(repository.RepositoryUri)) {
			fmt.Printf("Pushing [%s] to ECR (AWS)...\n", fullURI)

			ctx := context.Background()

			out, err := cli.ImagePush(ctx, fullURI, types.ImagePushOptions{RegistryAuth: aws.StringValue(authStr)})
			if err != nil {
				panic(err)
			}
			defer out.Close()

			termFd, isTerm := term.GetFdInfo(os.Stdout)
			if err := jsonmessage.DisplayJSONMessagesStream(out, os.Stdout, termFd, isTerm, nil); err != nil {
				return false
			}
			return true
		}
	}
	return false
}
