package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type (
	CloudFront struct {
		CloudfrontID      string `yaml:"id"`
		CloudFrontPattern string `yaml:"pattern,omitempty"`
		IgnoreDeploy      bool   `yaml:"ignore,omitempty"`
		AssociatedService string `yaml:"service,omitempty"`
	}

	CloudFronts struct {
		CloudFronts []CloudFront
	}
)

func LoadCloudFront(cloudfronts *CloudFronts, env *string) error {
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

	decoder.KnownFields(false)

	err = decoder.Decode(cloudfronts)
	if err != nil {
		log.Fatal(fmt.Errorf("Cloudfront:  Could not decode config file %s\n%v\n", configFilePath, err))
	}

	return nil
}
