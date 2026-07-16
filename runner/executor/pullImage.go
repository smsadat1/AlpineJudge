// prepares command to execute and runs the container
package executor

import (
	"context"
	"log"

	containerd "github.com/containerd/containerd"
	"github.com/containerd/containerd/errdefs"
)

// check image cache first then download image if cache miss
func getContainerImage(imageName string, client *containerd.Client, ctx context.Context) containerd.Image {

	image, err := client.GetImage(ctx, imageName)

	if err == nil {
		log.Printf("Image: %v found locally, skipping download\n", imageName)
		return image
	} else if errdefs.IsNotFound(err) {
		log.Printf("Image: %v not found locally, downloading image...\n", imageName)
		// download image
		image, err := client.Pull(ctx, imageName, containerd.WithPullUnpack)
		if err != nil {
			return nil
		}
		log.Printf("Successfully downloaded and pulled image: %s\n", image.Name())
	} else {
		log.Printf("Unexpected error occured querying image %v", err)
		return nil
	}

	return image
}

// func startContainer(rules utils.ExecRules) error {
// 	// init client and setup namespace
// 	client, err := containerd.New("/run/containerd/containerd.sock")
// 	if err != nil {
// 		log.Printf("[nsrunner] Failed to create containerd: %v", err)
// 		return err
// 	}
// 	defer client.Close()
// 	ctx := namespaces.WithNamespace(context.Background(), "alpine_judge")

// 	// spawn container
// 	container, err := spawnContainer(ctx, client, rules)
// 	if err != nil {
// 		log.Printf("Failed to spawn container: %v", err)
// 		return err
// 	}

// 	// delete container w/snapshot when main() exits
// 	defer container.Delete(ctx, containerd.WithSnapshotCleanup)

// 	// create and manage container
// 	err = createAndManageTask(container, ctx, rules)
// 	if err != nil {
// 		log.Printf("Container task management failed %v", err)
// 		return err
// 	}

// 	return nil
// }
