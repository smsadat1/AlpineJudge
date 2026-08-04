package internal

import (
	"fmt"
	"math/rand"

	"github.com/jaevor/go-nanoid"
)

// creates unique contianerID because same containerID under same namespace creates collision
func generateContainerID() string {
	nanoID, err := nanoid.CustomASCII("0123456789", 12)
	contID := nanoID()
	if err != nil {
		// fallback to random numbers
		contID = fmt.Sprint(rand.Intn(3000000))
	}
	return "ajcont-" + contID
}
