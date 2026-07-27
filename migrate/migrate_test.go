package migrate_test

import (
	"testing"

	msmigrate "github.com/opentoxic/microscope/migrate"
)

func TestSource(t *testing.T) {
	source, err := msmigrate.Source()
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()

	version, err := source.First()
	if err != nil {
		t.Fatal(err)
	}
	if version != 1 {
		t.Fatalf("first version %d", version)
	}
}
