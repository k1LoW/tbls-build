package cmd

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var cases = []string{"case_config", "case_schema", "case_postgres_schema", "case_prune"}

func TestRunBuildFile(t *testing.T) {
	for _, c := range cases {
		os.Setenv("TBLS_SCHEMA", "")
		os.Setenv("TBLS_CONFIG_PATH", "")

		dir := filepath.Join(testdataDir(), c)

		// TBLS_SCHEMA
		{
			path := filepath.Join(dir, "schema.json.env")
			if _, err := os.Stat(path); err == nil {
				b, err := ioutil.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				os.Setenv("TBLS_SCHEMA", string(b))
			}
		}

		// TBLS_CONFIG_PATH
		{
			path := filepath.Join(dir, "tbls.yml.env")
			if _, err := os.Stat(path); err == nil {
				os.Setenv("TBLS_CONFIG_PATH", path)
			}
		}

		// underlays
		underlays, err := filepath.Glob(filepath.Join(dir, "underlays", "*"))
		if err != nil {
			t.Fatal(err)
		}

		// overlays
		overlays, err := filepath.Glob(filepath.Join(dir, "overlays", "*"))
		if err != nil {
			t.Fatal(err)
		}

		// got
		stdout := new(bytes.Buffer)
		if err := runBuild(underlays, overlays, stdout); err != nil {
			t.Fatal(err)
		}
		got := stdout.Bytes()

		// want
		want, err := ioutil.ReadFile(filepath.Join(dir, "out.yml.golden"))
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(string(got), string(want), nil); diff != "" {
			t.Errorf("case '%s': diff exists\n%s", c, diff)
		}
	}
}

func TestRunBuildDir(t *testing.T) {
	for _, c := range cases {
		os.Setenv("TBLS_SCHEMA", "")
		os.Setenv("TBLS_CONFIG_PATH", "")

		dir := filepath.Join(testdataDir(), c)

		// TBLS_SCHEMA
		{
			path := filepath.Join(dir, "schema.json.env")
			if _, err := os.Stat(path); err == nil {
				b, err := ioutil.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				os.Setenv("TBLS_SCHEMA", string(b))
			}
		}

		// TBLS_CONFIG_PATH
		{
			path := filepath.Join(dir, "tbls.yml.env")
			if _, err := os.Stat(path); err == nil {
				os.Setenv("TBLS_CONFIG_PATH", path)
			}
		}

		// underlays dir
		underlays := []string{filepath.Join(dir, "underlays")}

		// overlays dir
		overlays := []string{filepath.Join(dir, "overlays")}

		// got
		stdout := new(bytes.Buffer)
		if err := runBuild(underlays, overlays, stdout); err != nil {
			t.Fatal(err)
		}
		got := stdout.Bytes()

		// want
		want, err := ioutil.ReadFile(filepath.Join(dir, "out.yml.golden"))
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(string(got), string(want), nil); diff != "" {
			t.Errorf("case '%s': diff exists\n%s", c, diff)
		}
	}
}

func testdataDir() string {
	wd, _ := os.Getwd()
	dir, _ := filepath.Abs(filepath.Join(filepath.Dir(wd), "testdata"))
	return dir
}
