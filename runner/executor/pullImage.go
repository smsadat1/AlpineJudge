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
