package scheduler

import "github.com/containerd/containerd"

type ContainerQueueChan struct {
	In  chan containerd.Container
	Out chan containerd.Container
}

func InitContainerQueue() *ContainerQueueChan {

	cq := &ContainerQueueChan{
		In:  make(chan containerd.Container),
		Out: make(chan containerd.Container),
	}

	go func() {
		defer close(cq.Out)
		var queue []containerd.Container // hold excess warm containers dynamically

		for {
			var outChan chan containerd.Container
			var nextContainer containerd.Container

			if len(queue) > 0 {
				outChan = cq.Out
				nextContainer = queue[0]
			}

			select {
			// accept more containers
			case container, ok := <-cq.In:
				if !ok {
					// input channel close, push remaining containers
					for _, cont := range queue {
						cq.Out <- cont
					}
					return
				}
				queue = append(queue, container) // queue grows dynamicaly

			// send data to the output channel (only active if queue has items)
			case outChan <- nextContainer:
				queue = queue[1:] // Pop containers
			}
		}
	}()

	return cq
}
