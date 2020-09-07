package gtddocker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
)

func TagLocalDockerImageFrom(dockerImageSource, dockerNewImageTag string) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	if err := cli.ImageTag(ctx, dockerImageSource, dockerNewImageTag); err != nil {
		log.Fatal(err)
	}
}

func PullFromPrivateRegistry(authConfig *types.AuthConfig, dockerImageName string) bool {

	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		panic(err)
	}

	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	out, err := cli.ImagePull(ctx, dockerImageName, types.ImagePullOptions{RegistryAuth: authStr})
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	// add progressBar here
	termFd, isTerm := term.GetFdInfo(os.Stdout)
	if err := jsonmessage.DisplayJSONMessagesStream(out, os.Stdout, termFd, isTerm, nil); err != nil {
		return false
	}

	return true
}
