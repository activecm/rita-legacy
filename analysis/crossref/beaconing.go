package crossref

import "github.com/ocmdev/rita/database"

type (
	BeaconingSelector struct{}
)

func (s BeaconingSelector) GetName() string {
	return "beaconing"
}

func (s BeaconingSelector) Select(res *database.Resources) (<-chan string, <-chan string) {
	internalHosts := make(chan string)
	externalHosts := make(chan string)
	go func() {
		close(internalHosts)
		close(externalHosts)
	}()
	return internalHosts, externalHosts
}
