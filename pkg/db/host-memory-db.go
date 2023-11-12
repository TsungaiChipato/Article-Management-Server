package db

import (
	"fmt"

	"github.com/tryvium-travels/memongo"
)

type MockMongo struct {
	Close func()
}

func (mm *MockMongo) HostMemoryDb(mongodPath string) (string, error) {
	// Start a temporary MongoDB server using memongo.
	server, err := memongo.StartWithOptions(&memongo.Options{
		MongodBin: mongodPath,
	})
	if err != nil {
		return "", err
	}

	mm.Close = func() {
		server.Stop()
	}

	// Print the MongoDB server URI.
	fmt.Printf("Mongo deployed at: %s\n", server.URI())
	return server.URI(), nil
}
