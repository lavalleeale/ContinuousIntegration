package db

import (
	"context"
	"log"
	"os"

	"github.com/go-rel/migration"
	"github.com/go-rel/postgres"
	"github.com/go-rel/rel"
	"github.com/lavalleeale/ContinuousIntegration/db/migrations"
	_ "github.com/lib/pq"
)

var Db rel.Repository

var adapter rel.Adapter

func Open() error {
	adapter, err := postgres.Open(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		return err
	}

	// initialize rel's repo.
	Db = rel.New(adapter)

	Db.Instrumentation(func(ctx context.Context, op string, message string) func(err error) {
		return func(error) {}
	})

	m := migration.New(Db)

	m.Register(0, migrations.MigrateCreateUsers, migrations.RollbackCreateUsers)

	m.Migrate(context.TODO())

	return nil
}

func Close() {
	adapter.Close()
}
