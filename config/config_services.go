package config

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/service/ecs"
	"gopkg.in/yaml.v3"
)

type (
	Label struct {
		Key   string `yaml:"key"`
		Value string `yaml:"value"`
	}

	Service struct {
		Name                 string  `yaml:"name"`
		Registry             string  `yaml:"registry"`
		Provider             string  `yaml:"provider,omitempty"`
		IgnoreDeploy         bool    `yaml:"ignore,omitempty"`
		UpdateECR            string  `yaml:"update_ecr,omitempty"`
		UpdateChildTask      bool    `yaml:"update_child_task,omitempty"`
		Labels               []Label `yaml:"labels,omitempty"`
		TaskExecutionRoleArn string  `yaml:"task_execution_role_arn,omitempty"`
		TaskRoleArn          string  `yaml:"task_role_arn,omitempty"`
		TaskARN              string
		Status               string
		RunningCount         int64
		TaskDefinition       *ecs.TaskDefinition
	}

	Services struct {
		Github     string `yaml:"github,omitempty"`
		ECSCluster string `yaml:"ecs_cluster"`
		ECSRegion  string `yaml:"ecs_region"`
		Services   []Service
	}

	Repository struct {
		Name           string `yaml:"name"`
		RepositoryName string `yaml:"repository_name"`
		Provider       string `yaml:"provider,omitempty"`
		IgnoreDeploy   bool   `yaml:"ignore,omitempty"`
		RegistryId     string
		RepositoryUri  string
	}

	Repositories struct {
		Repositories []Repository
	}

	ChildTask struct {
		Name          string `yaml:"name"`
		ParentService string `yaml:"parent"`
		IgnoreDeploy  bool   `yaml:"ignore,omitempty"`
	}

	ChildTasks struct {
		ChildTasks []ChildTask
	}
)

func LoadService(services *Services, repositories *Repositories, childtasks *ChildTasks, env *string) error {
	var configFilePath string

	configFilePath = fmt.Sprintf("gtd/%s.yaml", *env)
	if isExists, _ := exists(configFilePath); !isExists {
		configFilePath = fmt.Sprintf("configs/%s.yaml", *env)
	}

	f, err := os.Open(configFilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)

	//Reject invalid or unknow fields
	decoder.KnownFields(false)

	err = decoder.Decode(services)
	if err != nil {
		log.Fatal(fmt.Errorf("service:  could not decode config file %s: %v", configFilePath, err))
	}

	_, _ = f.Seek(0, io.SeekStart)
	decoder = yaml.NewDecoder(f)
	err = decoder.Decode(repositories)
	if err != nil {
		log.Fatal(fmt.Errorf("service:  could not decode config file %s: %v", configFilePath, err))
	}

	_, _ = f.Seek(0, io.SeekStart)
	decoder = yaml.NewDecoder(f)
	err = decoder.Decode(childtasks)
	if err != nil {
		log.Fatal(fmt.Errorf("service:  could not decode config file %s: %v", configFilePath, err))
	}

	return nil
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}
