package migrations

import (
	"embed"
	"io/fs"
)

//go:embed events/*.sql projections/*.sql artifacts/*.sql
var allFS embed.FS

var (
	// EventsFS contains the journal migrations.
	EventsFS = mustSub("events")
	// ProjectionsFS contains the projection store migrations.
	ProjectionsFS = mustSub("projections")
	// ArtifactsFS contains the artifact store migrations.
	ArtifactsFS = mustSub("artifacts")
)

func mustSub(dir string) fs.FS {
	sub, err := fs.Sub(allFS, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
