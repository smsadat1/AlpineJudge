// prepares command to execute and runs the container
package internal

import (
	"context"
	"fmt"
	"log"

	containerd "github.com/containerd/containerd"
	"github.com/containerd/containerd/errdefs"
)

// check image cache first then download image if cache miss
func getContainerImage(imageName string, client *containerd.Client, ctx context.Context) (containerd.Image, error) {

	image, err := client.GetImage(ctx, imageName)

	if err == nil {
		return image, nil
	}

	if errdefs.IsNotFound(err) {
		log.Printf("Image: %v not found locally, downloading image...\n", imageName)
		// download image
		pulledImage, err := client.Pull(ctx, imageName, containerd.WithPullUnpack)
		if err != nil {
			return nil, fmt.Errorf("failed to pull image %s: %w", imageName, err)
		}
		log.Printf("Successfully downloaded and pulled image: %s\n", image.Name())
		return pulledImage, nil
	}

	log.Printf("Unexpected error occured querying image %v", err)
	return nil, err
}
