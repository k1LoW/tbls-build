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
	"path/filepath"
	"reflect"

	"github.com/goccy/go-yaml"
	"github.com/imdario/mergo"
	"github.com/k1LoW/tbls/config"
	"github.com/k1LoW/tbls/datasource"
	"github.com/k1LoW/tbls/schema"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	underlays []string
	overlays  []string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tbls-build",
	Short: "build tbls config",
	Long:  `build tbls config.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := runBuild(underlays, overlays, os.Stdout)
		if err != nil {
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

	c, err := config.New()
	if err != nil {
		return err
	}

	// underlays
	for _, u := range underlays {
		uc, err := loadConfigFile(u)
		if err != nil {
			return err
		}
		uc, err = pruneConfig(uc, s)
		if err != nil {
			return err
		}
		c, err = mergeConfig(c, uc)
		if err != nil {
			return err
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
	c, err = mergeConfig(c, cc)
	if err != nil {
		return err
	}

	// overlays
	for _, o := range overlays {
		oc, err := loadConfigFile(o)
		if err != nil {
			return err
		}
		oc, err = pruneConfig(oc, s)
		if err != nil {
			return err
		}
		c, err = mergeConfig(c, oc)
		if err != nil {
			return err
		}
	}

	d := yaml.NewEncoder(stdout)
	defer d.Close()
	if err := d.Encode(c); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

type fileType int

const (
	fileTypeUnknown fileType = iota
	fileTypeConfig
	fileTypeSchema
)

func detectConfigOrSchema(f string) fileType {
	ext := filepath.Ext(f)
	switch ext {
	case ".yml", ".yaml":
		return fileTypeConfig
	case ".json":
		return fileTypeSchema
	default:
		return fileTypeUnknown
	}
}

func loadConfigFile(path string) (*config.Config, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	f, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if f.IsDir() {
		return nil, fmt.Errorf("%s is directory", path)
	}
	c, err := config.New()
	if err != nil {
		return nil, err
	}
	ft := detectConfigOrSchema(path)
	switch ft {
	case fileTypeUnknown:
		return nil, fmt.Errorf("unknown file type: %s", path)
	case fileTypeConfig:
		if err := c.LoadConfigFile(path); err != nil {
			return nil, err
		}
	case fileTypeSchema:
		s, err := datasource.AnalyzeJSONString("json://" + path)
		if err != nil {
			return nil, err
		}
		c, err = schemaToConfig(s)
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

func schemaToConfig(s *schema.Schema) (*config.Config, error) {
	cfg, err := config.New()
	if err != nil {
		return nil, err
	}
	for _, t := range s.Tables {
		ac := config.AdditionalComment{
			Table:              t.Name,
			TableComment:       t.Comment,
			ColumnComments:     map[string]string{},
			IndexComments:      map[string]string{},
			ConstraintComments: map[string]string{},
			TriggerComments:    map[string]string{},
			Labels:             []string{},
		}
		for _, c := range t.Columns {
			if c.Comment == "" {
				continue
			}
			ac.ColumnComments[c.Name] = c.Comment
		}
		for _, i := range t.Indexes {
			if i.Comment == "" {
				continue
			}
			ac.IndexComments[i.Name] = i.Comment
		}
		for _, c := range t.Constraints {
			if c.Comment == "" {
				continue
			}
			ac.ConstraintComments[c.Name] = c.Comment
		}
		for _, trig := range t.Triggers {
			if trig.Comment == "" {
				continue
			}
			ac.TriggerComments[trig.Name] = trig.Comment
		}
		for _, l := range t.Labels {
			ac.Labels = append(ac.Labels, l.Name)
		}
		cfg.Comments = append(cfg.Comments, ac)
	}

	for _, r := range s.Relations {
		ar := config.AdditionalRelation{
			Table:         r.Table.Name,
			Columns:       []string{},
			ParentTable:   r.ParentTable.Name,
			ParentColumns: []string{},
			Def:           r.Def,
		}
		for _, c := range r.Columns {
			ar.Columns = append(ar.Columns, c.Name)
		}
		for _, c := range r.ParentColumns {
			ar.ParentColumns = append(ar.ParentColumns, c.Name)
		}
		cfg.Relations = append(cfg.Relations, ar)
	}

	return cfg, nil
}

func pruneConfig(c *config.Config, s *schema.Schema) (*config.Config, error) {
	// TODO
	return c, nil
}

type commentsTransformer struct{}

func (t commentsTransformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	if typ == reflect.TypeOf([]config.AdditionalComment{}) {
		return func(dst, src reflect.Value) error {
			if dst.CanSet() {
				dstv, ok := dst.Interface().([]config.AdditionalComment)
				if !ok {
					return errors.New("transform error")
				}
				srcv, ok := src.Interface().([]config.AdditionalComment)
				if !ok {
					return errors.New("transform error")
				}
				a := srcv[:]
				a = append(a, dstv...)
				b := []config.AdditionalComment{}
				m := map[string]config.AdditionalComment{}
				for _, v := range a {
					key := v.Table
					if ac, ok := m[key]; ok {
						// tableComment
						if v.TableComment != "" {
							ac.TableComment = v.TableComment
						}
						// columnComments
						for k, c := range v.ColumnComments {
							ac.ColumnComments[k] = c
						}
						// indexComments
						for k, c := range v.IndexComments {
							ac.IndexComments[k] = c
						}
						// constraintComments
						for k, c := range v.ConstraintComments {
							ac.ConstraintComments[k] = c
						}
						// triggerComments
						for k, c := range v.TriggerComments {
							ac.TriggerComments[k] = c
						}
						// labels
						ac.Labels = uniq(append(ac.Labels, v.Labels...))

						m[key] = ac
					} else {
						m[key] = v
					}
				}
				for _, v := range a {
					key := v.Table
					if ac, ok := m[key]; ok {
						b = append(b, ac)
						delete(m, key)
					}
				}
				dst.Set(reflect.ValueOf(b))
			}
			return nil
		}
	}
	return nil
}

type relationsTransformer struct{}

func (t relationsTransformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	if typ == reflect.TypeOf([]config.AdditionalRelation{}) {
		return func(dst, src reflect.Value) error {
			if dst.CanSet() {
				dstv, ok := dst.Interface().([]config.AdditionalRelation)
				if !ok {
					return errors.New("transform error")
				}
				srcv, ok := src.Interface().([]config.AdditionalRelation)
				if !ok {
					return errors.New("transform error")
				}
				a := srcv[:]
				a = append(a, dstv...)
				b := []config.AdditionalRelation{}
				m := map[string]struct{}{}
				for _, v := range a {
					key := fmt.Sprintf("%s-%s-%s-%s", v.Table, v.Columns, v.ParentTable, v.ParentColumns)
					if _, ok := m[key]; !ok {
						m[key] = struct{}{}
						b = append(b, v)
					}
				}
				dst.Set(reflect.ValueOf(b))
			}
			return nil
		}
	}
	return nil
}

func uniq(a []string) []string {
	m := map[string]struct{}{}
	for _, e := range a {
		m[e] = struct{}{}
	}
	u := []string{}
	for _, e := range a {
		if _, ok := m[e]; ok {
			u = append(u, e)
			delete(m, e)
		}
	}
	return u
}

func mergeConfig(a, b *config.Config) (*config.Config, error) {
	err := mergo.Merge(a, *b, mergo.WithOverride, mergo.WithTransformers(commentsTransformer{}))
	return a, err
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringSliceVarP(&underlays, "underlay", "u", []string{}, "underlay")
	rootCmd.Flags().StringSliceVarP(&overlays, "overlay", "o", []string{}, "overlay")
	rootCmd.Flags().StringVarP(&out, "out", "", "", "output file path")
}
