package cmd

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRunBuild(t *testing.T) {
	cases := []string{"case_config"}
	for _, c := range cases {
		dir := filepath.Join(testdataDir(), c)

		// TBLS_SCHEMA
		{
			path := filepath.Join(dir, "schema.json")
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
			path := filepath.Join(dir, "tbls.yml")
			if _, err := os.Stat(path); err == nil {
				os.Setenv("TBLS_CONFIG_PATH", path)
			}
		}

		// underlays
		underlays, err := filepath.Glob(filepath.Join(dir, "underlay*"))
		if err != nil {
			t.Fatal(err)
		}

		// overlays
		overlays, err := filepath.Glob(filepath.Join(dir, "overlay*"))
		if err != nil {
			t.Fatal(err)
		}

		// got
		stdout := new(bytes.Buffer)
		err = runBuild(underlays, overlays, stdout)
		if err != nil {
			t.Fatal(err)
		}
		got := stdout.Bytes()

		// want
		want, err := ioutil.ReadFile(filepath.Join(dir, "out.yml.golden"))
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(string(got), string(want), nil); diff != "" {
			t.Errorf("diff\n%s", diff)
		}
	}
}

func testdataDir() string {
	wd, _ := os.Getwd()
	dir, _ := filepath.Abs(filepath.Join(filepath.Dir(wd), "testdata"))
	return dir
}
