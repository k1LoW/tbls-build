/*
Copyright © 2020 Ken'ichiro Oyama <k1lowxb@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/k1LoW/tbls-build/builder"
	"github.com/k1LoW/tbls/config"
	"github.com/k1LoW/tbls/datasource"
	"github.com/k1LoW/tbls/schema"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	underlays []string
	overlays  []string
	out       string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tbls-build",
	Short: "tbls-build is an external subcommand of tbls for customizing config file of tbls",
	Long:  `tbls-build is an external subcommand of tbls for customizing config file of tbls using other tbls.yml or schema.json.`,
	Run: func(cmd *cobra.Command, args []string) {
		var (
			o   *os.File
			err error
		)
		if out == "" {
			o = os.Stdout
		} else {
			o, err = os.Create(out)
			if err != nil {
				cmd.PrintErrln(err)
				os.Exit(1)
			}
		}
		if err := runBuild(underlays, overlays, o); err != nil {
			cmd.PrintErrln(err)
			os.Exit(1)
		}
	},
}

func runBuild(underlays, overlays []string, stdout io.Writer) error {
	var (
		s   *schema.Schema
		err error
	)
	sstr := os.Getenv("TBLS_SCHEMA")
	if sstr != "" {
		s, err = datasource.AnalyzeJSONString(os.Getenv("TBLS_SCHEMA"))
		if err != nil {
			return err
		}
	}

	b := builder.New(s)

	c, err := config.New()
	if err != nil {
		return err
	}

	// underlays
	for _, u := range underlays {
		paths, err := builder.LoadPatchFiles(u)
		if err != nil {
			return err
		}
		for _, p := range paths {
			uc, err := b.LoadPatchFile(p)
			if err != nil {
				return err
			}
			uc, err = b.PruneConfig(uc)
			if err != nil {
				return err
			}
			c, err = b.MergeConfig(c, uc)
			if err != nil {
				return err
			}
		}
	}

	// -c
	cc, err := config.New()
	if err != nil {
		return err
	}
	configPath := os.Getenv("TBLS_CONFIG_PATH")
	if configPath != "" {
		if err := cc.LoadConfigFile(configPath); err != nil {
			return err
		}
	}
	cc, err = b.PruneConfig(cc)
	if err != nil {
		return err
	}
	c, err = b.MergeConfig(c, cc)
	if err != nil {
		return err
	}

	// overlays
	for _, o := range overlays {
		paths, err := builder.LoadPatchFiles(o)
		if err != nil {
			return err
		}
		for _, p := range paths {
			oc, err := b.LoadPatchFile(p)
			if err != nil {
				return err
			}
			oc, err = b.PruneConfig(oc)
			if err != nil {
				return err
			}
			c, err = b.MergeConfig(c, oc)
			if err != nil {
				return err
			}
		}
	}

	d := yaml.NewEncoder(stdout)
	defer d.Close()
	if err := d.Encode(c); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringSliceVarP(&underlays, "underlay", "u", []string{}, "patch file or directory for underlaying")
	rootCmd.Flags().StringSliceVarP(&overlays, "overlay", "o", []string{}, "patch file or directory for overlaying")
	rootCmd.Flags().StringVarP(&out, "out", "", "", "output file path")
}
