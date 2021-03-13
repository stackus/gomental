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
	Pkgs       int
	Files      int
	Lines      int
	Globals    int
	Consts     int
	Interfaces int
	Structs    int
	Others     int
	Methods    int
	Funcs      int
}

type mentalSort []MentalCtx

var skip = map[string]struct{}{
	".git":    {},
	".github": {},
	".idea":   {},
	".vscode": {},
	"vendor":  {},
}

var userMaxDepth = 1
var userSkip []string
var userNoZero = false
var userTests = false

const tableFormat = `Path	Packages	Files	Lines	Global Vars	Constants	Interfaces	Structs	Other Types	Methods	Funcs
{{ range . }}{{ .Path }}	{{ .Pkgs }}	{{ .Files }}	{{ .Lines }}	{{ .Globals }}	{{ .Consts }}	{{ .Interfaces }}	{{ .Structs }}	{{ .Others }}	{{ .Methods }}	{{ .Funcs }}
{{ end }}
`

var rootCmd = &cobra.Command{
	Use:   "gomental path",
	Short: "Displays details about the golang source at the given path",
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
	rootCmd.Flags().IntVarP(&userMaxDepth, "depth", "d", userMaxDepth, `Display an entry for all directories "depth" directories deep`)
	rootCmd.Flags().StringSliceVarP(&userSkip, "skip", "s", userSkip, "Directory names to skip. format: dir,dir")
	rootCmd.Flags().BoolVar(&userNoZero, "no-zero", userNoZero, "Ignore golang source free directories")
	rootCmd.Flags().BoolVar(&userTests, "with-tests", userTests, "Include test files")

	rootCmd.Flags().Lookup("no-zero").NoOptDefVal = "true"
	rootCmd.Flags().Lookup("with-tests").NoOptDefVal = "true"
}

func runRoot(_ *cobra.Command, args []string) {
	var err error

	if userMaxDepth < 0 {
		userMaxDepth = 1
	}

	if userMaxDepth > 999 {
		userMaxDepth = 999
	}

	if !userTests {
		userSkip = append(userSkip, "testdata")
	}

	for _, s := range userSkip {
		if _, exists := skip[s]; !exists {
			skip[s] = struct{}{}
		}
	}

	rootPath := filepath.Clean(args[0]) + string(filepath.Separator)

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

		parts := strings.SplitN(strings.TrimPrefix(path, rootPath), string(filepath.Separator), userMaxDepth+1)

		depth := userMaxDepth
		if len(parts) < depth {
			depth = len(parts)
		}

		workingPath := string(filepath.Separator) + filepath.Join(parts[0:depth]...)
		if _, exists := mentalMap[workingPath]; !exists {
			mentalMap[workingPath] = &MentalCtx{
				Path: workingPath,
			}
		}

		var ctx MentalCtx

		ctx, err = parseDir(path)
		if err != nil {
			return err
		}

		mentalMap[workingPath].sum(ctx)

		return nil
	})
	if err != nil {
		fmt.Printf("error walking the path %s: %s\n", rootPath, err)
		return
	}

	mentalSlice := make([]MentalCtx, 0, len(mentalMap))
	for _, ctx := range mentalMap {
		if userNoZero && ctx.Files == 0 {
			continue
		}
		mentalSlice = append(mentalSlice, *ctx)
	}

	sort.Sort(mentalSort(mentalSlice))

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
		if userTests {
			return true
		}
		return !strings.HasSuffix(info.Name(), "test.go")
	}, parser.ParseComments)
	if err != nil {
		return ctx, err
	}

	for _, pkg := range pkgs {
		ctx.Pkgs++
		ctx.Files += len(pkg.Files)
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				switch d := decl.(type) {
				case *ast.GenDecl:
					switch d.Tok.String() {
					case "var":
						for _, spec := range d.Specs {
							if _, ok := spec.(*ast.ValueSpec); ok {
								ctx.Globals++
							}
						}
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
								fmt.Printf("unhandled type spec %T\n", s)
							}
						}
					case "const":
						for _, spec := range d.Specs {
							if _, ok := spec.(*ast.ValueSpec); ok {
								ctx.Consts++
							}
						}
					}
				case *ast.FuncDecl:
					if d.Recv == nil {
						ctx.Funcs++
					} else {
						ctx.Methods++
					}
				default:
					fmt.Printf("unhandled decl %T\n", d)
				}
			}
			f := fset.File(file.Pos())
			ctx.Lines += f.LineCount()
		}
	}

	return ctx, nil
}

func (c *MentalCtx) sum(other MentalCtx) {
	c.Pkgs += other.Pkgs
	c.Lines += other.Lines
	c.Files += other.Files
	c.Globals += other.Globals
	c.Consts += other.Consts
	c.Interfaces += other.Interfaces
	c.Structs += other.Structs
	c.Others += other.Others
	c.Methods += other.Methods
	c.Funcs += other.Funcs
}

func (x mentalSort) Len() int { return len(x) }
func (x mentalSort) Less(i, j int) bool {
	iPath := strings.Replace(x[i].Path, string(filepath.Separator), "\x00", -1)
	jPath := strings.Replace(x[j].Path, string(filepath.Separator), "\x00", -1)

	return iPath < jPath
}
func (x mentalSort) Swap(i, j int) { x[i], x[j] = x[j], x[i] }
