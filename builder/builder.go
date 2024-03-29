package builder

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/imdario/mergo"
	"github.com/k1LoW/tbls/config"
	"github.com/k1LoW/tbls/datasource"
	"github.com/k1LoW/tbls/schema"
)

type fileType int

const (
	fileTypeUnknown fileType = iota
	fileTypeConfig
	fileTypeSchema
)

type Builder struct {
	schema *schema.Schema
}

// New
func New(s *schema.Schema) *Builder {
	return &Builder{
		schema: s,
	}
}

func LoadPatchFiles(p string) ([]string, error) {
	paths := []string{}
	d, err := os.Stat(p)
	if err != nil {
		return paths, err
	}
	if d.IsDir() {
		files, err := filepath.Glob(filepath.Join(p, "*"))
		if err != nil {
			return paths, err
		}
		sort.Slice(files, func(i, j int) bool { return filepath.Base(files[i]) < filepath.Base(files[j]) })
		for _, f := range files {
			if detectConfigOrSchema(f) != fileTypeUnknown {
				paths = append(paths, f)
			}
		}
	} else {
		paths = []string{p}
	}
	return paths, nil
}

func (b *Builder) LoadPatchFile(path string) (*config.Config, error) {
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
		s, err := datasource.AnalyzeJSON("json://" + path)
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

func (b *Builder) PruneConfig(cfg *config.Config) (*config.Config, error) {
	if b.schema == nil {
		return cfg, nil
	}

	// prune default
	if cfg.DocPath == config.DefaultDocPath {
		cfg.DocPath = ""
	}
	if cfg.ER.Format == config.DefaultERFormat {
		cfg.ER.Format = ""
	}
	if cfg.ER.Distance != nil && *cfg.ER.Distance == config.DefaultERDistance {
		cfg.ER.Distance = nil
	}

	// normalize table name and prune non-existent table
	comments := []config.AdditionalComment{}
	for _, c := range cfg.Comments {
		t, err := b.schema.FindTableByName(c.Table)
		if err != nil {
			continue
		}
		c.Table = b.schema.NormalizeTableName(c.Table)
		for n := range c.ColumnComments {
			if _, err := t.FindColumnByName(n); err != nil {
				delete(c.ColumnComments, n)
			}
		}

		comments = append(comments, c)
	}
	cfg.Comments = comments

	relations := []config.AdditionalRelation{}
	for _, r := range cfg.Relations {
		if _, err := b.schema.FindTableByName(r.Table); err != nil {
			continue
		}
		if _, err := b.schema.FindTableByName(r.ParentTable); err != nil {
			continue
		}
		r.Table = b.schema.NormalizeTableName(r.Table)
		r.ParentTable = b.schema.NormalizeTableName(r.ParentTable)
		relations = append(relations, r)
	}
	cfg.Relations = relations

	return cfg, nil
}

type configTransformer struct{}

func (t configTransformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
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
				a := dstv[:]
				a = append(a, srcv...)
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

func (b *Builder) MergeConfig(dst, src *config.Config) (*config.Config, error) {
	err := mergo.Merge(dst, *src, mergo.WithOverride, mergo.WithTransformers(configTransformer{}))
	return dst, err
}

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

func schemaToConfig(s *schema.Schema) (*config.Config, error) {
	cfg, err := config.New()
	if err != nil {
		return nil, err
	}
	cfg.Name = s.Name
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
