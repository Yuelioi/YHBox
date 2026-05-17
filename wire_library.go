package main

import "yhbox/internal/services/container"

// libraryContainerAccessAdapter 把 *container.Store 适配到 library.containerStoreAccess 窄接口
type libraryContainerAccessAdapter struct {
	store *container.Store
}

func (a *libraryContainerAccessAdapter) Get(id string) (container.Container, bool) {
	return a.store.Get(id)
}

func (a *libraryContainerAccessAdapter) SaveSubgraph(containerID string, sg *container.Subgraph) error {
	return a.store.SaveSubgraph(containerID, sg)
}
