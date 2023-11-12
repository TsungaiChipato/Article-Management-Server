package db_test

import (
	"article-management-service/pkg/db"
	"article-management-service/pkg/env"
	"testing"
)

type MockMongo struct {
	Close func()
}

func TestMockMongo_HostMemoryDb(t *testing.T) {
	cfg, err := env.Load()
	if err != nil {
		panic(err)
	}

	t.Run("Successfully host memory mongo", func(t *testing.T) {
		mm := db.MockMongo{}
		_, err := mm.HostMemoryDb(cfg.MongodPath)
		if err != nil {
			t.Error("Failed to connect to memory server")
			t.FailNow()
		}
		defer mm.Close()
	})

}
