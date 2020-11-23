package cobra

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/gpkfr/goretdep/aws"
	"github.com/gpkfr/goretdep/config"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	awsProfile string
)

type (
	//Command Struct that bind all commands
	Command struct {
		cobra.Command

		Version             string
		AWSProfile          string
		GTenv               string
		SharedSecret        string
		SelectedServices    []string
		Services            config.Services
		Repositories        config.Repositories
		ChildTasks          config.ChildTasks
		CloudFronts         config.CloudFronts
		AWSSession          *aws.AWSSession
		DockerHubAuthConfig *types.AuthConfig
		TableStyle          string
		ShowTableIndexAbove int
	}
)

//CheckEnv populate the env used
//to determine the file who describe an Environment
func (cmd *Command) CheckEnv() {
	if strings.EqualFold("", cmd.GTenv) {
		cmd.GTenv = viper.GetString("default_env")

		if strings.EqualFold("", cmd.GTenv) {
			log.Println("ENV not set... Please use '--env'")
			os.Exit(1)
		}
	}

}

//GetAWSSession Instanciate a Global reusable AWSSession
func (cmd *Command) GetAWSSession() {
	var err error
	cmd.AWSSession, err = aws.NewAWSSession(&cmd.Services.ECSRegion, &cmd.AWSProfile)
	if err != nil {
		log.Fatal(err)
	}
}

//NewCommand Return Pointer to the Command struct{}
//and bind other command.
func NewCommand(version string) *Command {
	var cmd = &Command{
		Command: cobra.Command{
			Use:   "gtd",
			Short: "Tool to deploy AWS ECS Configuration",

			TraverseChildren: true,
		},
		Version: version,
	}

	var versionFlag bool
	var dockerHubPassword string

	cobra.OnInitialize(func() {
		if cfgFile != "" {
			viper.SetConfigFile(cfgFile)
		} else {
			home, err := homedir.Dir()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			viper.AddConfigPath(home)
			viper.SetConfigName(".gtd")

			viper.SetEnvPrefix("gtd")
			viper.AutomaticEnv() // read in environment variables that match
			// if a config file is found, read it in.
			if err := viper.ReadInConfig(); err == nil {
				fmt.Println("Using config file", viper.ConfigFileUsed())
			}

			if strings.EqualFold("", cmd.AWSProfile) {
				cmd.AWSProfile = viper.GetString("aws_profile")
			}
			cmd.TableStyle = viper.GetString("table_style")
			cmd.ShowTableIndexAbove = viper.GetInt("table_index_above")

			cmd.SharedSecret = viper.GetString("shared_secret")

			dockerHubLogin := viper.GetString("docker_login")

			secretDockerHubPassword := viper.GetString("secret_docker_password")

			if !strings.EqualFold("", cmd.SharedSecret) && len(cmd.SharedSecret) == 32 && !strings.EqualFold("", secretDockerHubPassword) {
				dockerHubPassword = Decrypt([]byte(secretDockerHubPassword), []byte(cmd.SharedSecret))
			} else {
				dockerHubPassword = viper.GetString("docker_password")
			}

			// fmt.Println(Decrypt([]byte(viper.GetString("mysecret")), []byte(sharedSecret)))

			if !strings.EqualFold("", dockerHubLogin) && !strings.EqualFold("", dockerHubPassword) {
				authConfig := &types.AuthConfig{
					Username: dockerHubLogin,
					Password: dockerHubPassword,
				}

				cmd.DockerHubAuthConfig = authConfig
			}
		}
	})

	cmd.Run = func(cobraCmd *cobra.Command, args []string) {
		if versionFlag {
			fmt.Printf("%s version %s\n", cmd.Name(), cmd.Version)
		} else {
			_ = cobraCmd.Usage()
		}

	}

	cmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Print version information")
	cmd.Flags().StringVar(&cmd.AWSProfile, "profile", "", "Profile AWS to use [--profile chaudron]")
	// cmd.Flags().StringVar(&cfgFile, "configFile", "", "use global config file '(default to $HOME/.gtd.yaml)'")
	if cmd.AWSProfile == "" {
		cmd.AWSProfile = awsProfile
	}

	NewDeployCommand(cmd)
	NewStatusCommand(cmd)
	NewInvalidateCommand(cmd)
	NewListInvalidationCommand(cmd)
	NewEncryptVarCommand(cmd)
	return cmd
}
