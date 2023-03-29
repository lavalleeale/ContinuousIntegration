package migrations

import (
	"github.com/go-rel/rel"
)

func MigrateCreateUsers(schema *rel.Schema) {
	schema.CreateTable("users", func(t *rel.Table) {
		t.ID("id")
		t.String("username")
		t.String("password")
		t.Int("organization_id")
	})

	schema.CreateTable("organizations", func(t *rel.Table) {
		t.ID("id")
	})

	schema.CreateTable("repos", func(t *rel.Table) {
		t.ID("id")
		t.String("url")
		t.Int("organization_id")
	})

	schema.CreateTable("builds", func(t *rel.Table) {
		t.ID("id")
		t.String("repo_id")
	})

	schema.CreateTable("containers", func(t *rel.Table) {
		t.ID("id")
		t.String("name")
		t.SmallInt("code")
		t.String("command", rel.Limit(8192))
		t.String("image")
		t.String("environment", rel.Limit(512))
		t.String("service_command", rel.Limit(512))
		t.String("service_healthcheck", rel.Limit(512))
		t.String("service_environment", rel.Limit(512))
		t.String("service_image")
		t.String("log", rel.Limit(25_000))
		t.Int("build_id")
	})
}

func RollbackCreateUsers(schema *rel.Schema) {
	schema.DropTable("builds")
	schema.DropTable("users")
	schema.DropTable("repos")
	schema.DropTable("organizations")
	schema.DropTable("containers")
}
