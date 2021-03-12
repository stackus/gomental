package cmd

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/spf13/cobra"
)

/*
	TODO other things to record
	avg Lines
	avg field count
	avg method count
	avg func count
	avg comment Lines
	ratio comment/code
*/

type MentalCtx struct {
	Path       string
	Files      int
	Globals    int
	Interfaces int
	Structs    int
	Others     int
	Methods    int
	Funcs      int
	Lines      int
}

var maxDepth = 1

// TODO accept string slice with --skip/-s
var skip = map[string]struct{}{
	".git":         {},
	".github":      {},
	".idea":        {},
	"node_modules": {},
	"vendor":       {},
	"testdata":     {},
}

const tableFormat = `Path	Files	Globals	Interfaces	Structs	Others	Methods	Funcs	Lines
{{ range . }}{{ .Path }}	{{ .Files }}	{{ .Globals }}	{{ .Interfaces }}	{{ .Structs }}	{{ .Others }}	{{ .Methods }}	{{ .Funcs }}	{{ .Lines }}
{{ end }}
`

var rootCmd = &cobra.Command{
	Use:   "gomental path",
	Short: "TODO short",
	Long:  `TODO long`,
	Args:  cobra.ExactArgs(1),
	Run:   runRoot,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().IntVarP(&maxDepth, "depth", "d", maxDepth, `Display an entry for all directories "depth" directories deep`)
}

func runRoot(_ *cobra.Command, args []string) {
	var err error

	rootPath := filepath.Clean(args[0])

	mentalMap := make(map[string]*MentalCtx)

	err = filepath.WalkDir(rootPath, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("issue accessing %s : %s\n", path, err)
			return err
		}

		if _, exists := skip[info.Name()]; exists {
			return filepath.SkipDir
		}

		if !info.IsDir() {
			return nil
		}

		parts := strings.SplitN(strings.TrimPrefix(path, rootPath), string(filepath.Separator), maxDepth+2)

		ctx, err := parseDir(path)
		if err != nil {
			return err
		}

		depth := maxDepth + 1
		if len(parts) < depth {
			depth = len(parts)
		}

		workingPath := string(filepath.Separator)
		for _, s := range parts[0:depth] {
			workingPath = filepath.Join(workingPath, s)
			if _, exists := mentalMap[workingPath]; !exists {
				mentalMap[workingPath] = &MentalCtx{
					Path: workingPath,
				}
			}
			mentalMap[workingPath].sum(ctx)
		}

		return nil
	})
	if err != nil {
		fmt.Printf("error walking the Path %q: %v\n", rootPath, err)
		return
	}

	keys := make([]string, 0, len(mentalMap))
	for key := range mentalMap {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	mentalSlice := make([]*MentalCtx, 0, len(mentalMap))
	for _, key := range keys {
		mentalSlice = append(mentalSlice, mentalMap[key])
	}

	out := tabwriter.NewWriter(os.Stdout, 1, 0, 2, ' ', 0)
	tmpl, err := template.New("").Parse(tableFormat)
	if err != nil {
		fmt.Printf("error compiling template : %s", err)
		return
	}
	err = tmpl.Execute(out, mentalSlice)
	if err != nil {
		fmt.Printf("error filling template : %s", err)
		return
	}
	err = out.Flush()
	if err != nil {
		fmt.Printf("error outputting table : %s", err)
		return
	}
}

func parseDir(path string) (MentalCtx, error) {
	ctx := MentalCtx{}

	fset := token.NewFileSet()

	pkgs, err := parser.ParseDir(fset, path, func(info fs.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "test.go")
	}, parser.ParseComments)
	if err != nil {
		return ctx, err
	}

	for _, pkg := range pkgs {
		ctx.Files += len(pkg.Files)
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				switch d := decl.(type) {
				case *ast.GenDecl:
					switch d.Tok.String() {
					case "var":
						ctx.Globals++
					case "type":
						for _, spec := range d.Specs {
							switch s := spec.(type) {
							case *ast.TypeSpec:
								switch s.Type.(type) {
								case *ast.InterfaceType:
									ctx.Interfaces++
								case *ast.StructType:
									ctx.Structs++
								default:
									ctx.Others++
								}
							default:
								fmt.Printf("%v %T\n", s, s)
							}
						}
					case "import": // ignored
					}
				case *ast.FuncDecl:
					if d.Recv == nil {
						ctx.Funcs++
					} else {
						ctx.Methods++
					}
				default:
					fmt.Printf("unknown decl %v %T\n", d, d)
				}
			}
			f := fset.File(file.Pos())
			ctx.Lines += f.LineCount()
		}
	}

	return ctx, nil
}

func (c *MentalCtx) sum(other MentalCtx) {
	c.Files += other.Files
	c.Globals += other.Globals
	c.Interfaces += other.Interfaces
	c.Structs += other.Structs
	c.Others += other.Others
	c.Methods += other.Methods
	c.Funcs += other.Funcs
	c.Lines += other.Lines
}
