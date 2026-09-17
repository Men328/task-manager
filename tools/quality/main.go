package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Config struct {
	Exclude       []string       `json:"exclude"`
	NoComments    []string       `json:"noComments"`
	FlatLayers    []string       `json:"flatLayers"`
	RequiredFiles []RequiredFile `json:"requiredFiles"`
	ImportRules   []ImportRule   `json:"importRules"`
	Cmd           CmdRules       `json:"cmd"`
}

type RequiredFile struct {
	Path                 string `json:"path"`
	MustDeclareInterface bool   `json:"mustDeclareInterface"`
}

type ImportRule struct {
	Layer  string   `json:"layer"`
	Forbid []string `json:"forbid"`
	Reason string   `json:"reason"`
}

type CmdRules struct {
	RequiredDirs   []string `json:"requiredDirs"`
	RequireImports []string `json:"requireImports"`
	ForbidImports  []string `json:"forbidImports"`
	ForbidPatterns []string `json:"forbidPatterns"`
}

type violation struct {
	rule string
	file string
	line int
	msg  string
}

type checker struct {
	cfg  Config
	root string
	vios []violation
}

var ruleOrder = []string{"import-rules", "flat-layers", "required-files", "no-comments", "cmd"}

func main() {
	configPath := flag.String("config", "quality.json", "đường dẫn file quality config")
	flag.Parse()

	cfg, root, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "quality: %v\n", err)
		os.Exit(2)
	}

	c := &checker{cfg: cfg, root: root}
	c.checkImports()
	c.checkFlatLayers()
	c.checkRequiredFiles()
	c.checkNoComments()
	c.checkCmd()

	fmt.Printf("quality: root=%s\n", root)

	counts := map[string]int{}
	for _, v := range c.vios {
		counts[v.rule]++
	}

	for _, rule := range ruleOrder {
		if counts[rule] == 0 {
			fmt.Printf("[PASS] %s\n", rule)
			continue
		}
		fmt.Printf("[FAIL] %s (%d)\n", rule, counts[rule])
		for _, v := range c.vios {
			if v.rule != rule {
				continue
			}
			loc := v.file
			if v.line > 0 {
				loc = fmt.Sprintf("%s:%d", v.file, v.line)
			}
			fmt.Printf("       %s: %s\n", loc, v.msg)
		}
	}

	if len(c.vios) == 0 {
		fmt.Println("quality: OK")
		return
	}
	fmt.Printf("quality: %d lỗi\n", len(c.vios))
	os.Exit(1)
}

func loadConfig(path string) (Config, string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return Config{}, "", err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return Config{}, "", err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, "", fmt.Errorf("parse %s: %w", abs, err)
	}
	return cfg, filepath.Dir(abs), nil
}

func (c *checker) add(rule, file string, line int, format string, args ...any) {
	c.vios = append(c.vios, violation{rule: rule, file: file, line: line, msg: fmt.Sprintf(format, args...)})
}

func (c *checker) rel(path string) string {
	r, err := filepath.Rel(c.root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(r)
}

func (c *checker) excluded(path string) bool {
	rel := c.rel(path)
	for _, ex := range c.cfg.Exclude {
		if rel == ex || strings.HasPrefix(rel, ex+"/") {
			return true
		}
	}
	return false
}

func (c *checker) services() []string {
	base := filepath.Join(c.root, "service")
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(base, e.Name())
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			out = append(out, dir)
		}
	}
	sort.Strings(out)
	return out
}

func allGoFiles(dir string) []string {
	var out []string
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			out = append(out, path)
		}
		return nil
	})
	sort.Strings(out)
	return out
}

func hasGoFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
			return true
		}
	}
	return false
}

func importsOf(fset *token.FileSet, path string) ([]string, map[string]int, error) {
	file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, nil, err
	}
	paths := make([]string, 0, len(file.Imports))
	lines := make(map[string]int, len(file.Imports))
	for _, imp := range file.Imports {
		p := strings.Trim(imp.Path.Value, `"`)
		paths = append(paths, p)
		if _, ok := lines[p]; !ok {
			lines[p] = fset.Position(imp.Pos()).Line
		}
	}
	return paths, lines, nil
}

func (c *checker) checkImports() {
	for _, svc := range c.services() {
		for _, rule := range c.cfg.ImportRules {
			dir := filepath.Join(svc, filepath.FromSlash(rule.Layer))
			for _, f := range allGoFiles(dir) {
				fset := token.NewFileSet()
				paths, lines, err := importsOf(fset, f)
				if err != nil {
					c.add("import-rules", c.rel(f), 0, "không parse được: %v", err)
					continue
				}
				for _, p := range paths {
					for _, bad := range rule.Forbid {
						if strings.Contains(p, bad) {
							c.add("import-rules", c.rel(f), lines[p], "%s (import %q)", rule.Reason, p)
						}
					}
				}
			}
		}
	}
}

func (c *checker) checkFlatLayers() {
	for _, svc := range c.services() {
		for _, layer := range c.cfg.FlatLayers {
			dir := filepath.Join(svc, filepath.FromSlash(layer))
			if _, err := os.Stat(dir); err != nil {
				continue
			}
			_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
				if err != nil || !d.IsDir() || path == dir {
					return nil
				}
				if hasGoFiles(path) {
					c.add("flat-layers", c.rel(path), 0, "tầng %s phải flat, không có package con", layer)
					return filepath.SkipDir
				}
				return nil
			})
		}
	}
}

func (c *checker) checkRequiredFiles() {
	for _, svc := range c.services() {
		for _, rf := range c.cfg.RequiredFiles {
			p := filepath.Join(svc, filepath.FromSlash(rf.Path))
			if _, err := os.Stat(p); err != nil {
				c.add("required-files", c.rel(p), 0, "thiếu file contract bắt buộc")
				continue
			}
			if !rf.MustDeclareInterface {
				continue
			}
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, p, nil, 0)
			if err != nil {
				c.add("required-files", c.rel(p), 0, "không parse được: %v", err)
				continue
			}
			if !declaresInterface(file) {
				c.add("required-files", c.rel(p), 0, "phải khai báo ít nhất 1 interface contract")
			}
		}
	}
}

func declaresInterface(f *ast.File) bool {
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if _, ok := ts.Type.(*ast.InterfaceType); ok {
				return true
			}
		}
	}
	return false
}

func (c *checker) checkNoComments() {
	for _, name := range c.cfg.NoComments {
		base := filepath.Join(c.root, filepath.FromSlash(name))
		_ = filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if path != base && c.excluded(path) {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				return nil
			}
			if len(file.Comments) == 0 {
				return nil
			}
			line := fset.Position(file.Comments[0].Pos()).Line
			c.add("no-comments", c.rel(path), line, "code Go không được có comment (%d nhóm)", len(file.Comments))
			return nil
		})
	}
}

func (c *checker) checkCmd() {
	for _, svc := range c.services() {
		for _, reqDir := range c.cfg.Cmd.RequiredDirs {
			dir := filepath.Join(svc, filepath.FromSlash(reqDir))
			files := allGoFiles(dir)
			if len(files) == 0 {
				c.add("cmd", c.rel(dir), 0, "thiếu thư mục %s", reqDir)
				continue
			}

			found := map[string]bool{}
			for _, f := range files {
				fset := token.NewFileSet()
				paths, lines, err := importsOf(fset, f)
				if err != nil {
					c.add("cmd", c.rel(f), 0, "không parse được: %v", err)
					continue
				}
				for _, p := range paths {
					found[p] = true
					for _, bad := range c.cfg.Cmd.ForbidImports {
						if p == bad {
							c.add("cmd", c.rel(f), lines[p], "cmd không được import %q; lifecycle do fx quản lý", p)
						}
					}
				}

				src, err := os.ReadFile(f)
				if err != nil {
					continue
				}
				for _, pat := range c.cfg.Cmd.ForbidPatterns {
					if strings.Contains(string(src), pat) {
						c.add("cmd", c.rel(f), 0, "cmd không được chứa %q; dùng fx lifecycle", pat)
					}
				}
			}

			for _, req := range c.cfg.Cmd.RequireImports {
				if !found[req] {
					c.add("cmd", c.rel(dir), 0, "cmd phải dùng %q cho DI + lifecycle", req)
				}
			}
		}
	}
}
