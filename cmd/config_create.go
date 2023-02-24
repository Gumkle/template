package cmd

import (
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"go/ast"
	"go/format"
	"go/printer"
	"go/token"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/tools/go/ast/astutil"
	"os"
	"strings"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new command set.",
	Long: `Create a new command set. For this set create new yaml file, new source file
and a new struct inside of this file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("unexpected length of argument list")
		}
		categoryName := strings.ToLower(args[0])
		if exists, err := categoryExists(categoryName); exists || err != nil {
			if err != nil {
				return err
			}
			return fmt.Errorf("category %s already exists", categoryName)
		}
		err := createCategory(categoryName)
		if err != nil {
			return err
		}
		return nil
	},
}

func createCategory(name string) error {
	// create yaml file
	configFilePath := fmt.Sprintf("config/%s.yaml", name)
	yaml, err := os.Create(configFilePath)
	if err != nil {
		return err
	}
	err = yaml.Close()
	if err != nil {
		return err
	}

	// begin AST
	fset := token.NewFileSet()
	structName := cases.Title(language.Und).String(name) + "Config"
	file := &ast.File{Name: &ast.Ident{Name: name}}

	// create struct
	typeDeclaration := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{Name: &ast.Ident{Name: structName}, Type: &ast.StructType{
				Fields: &ast.FieldList{
					Opening: token.NoPos,
					Closing: token.NoPos,
				}}}}}
	file.Decls = []ast.Decl{typeDeclaration}

	// create struct constructor
	errorHandlingStmt := &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  &ast.Ident{Name: "err"},
			Op: token.NEQ,
			Y:  ast.NewIdent("nil"),
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("nil"), &ast.Ident{Name: "err"}}},
		}},
	}
	funcDefinition := &ast.FuncDecl{
		Doc: &ast.CommentGroup{List: []*ast.Comment{
			{Text: fmt.Sprintf("// New%s unmarshalls yaml data to struct and returns a pointer to it", structName)},
		}},
		Name: &ast.Ident{Name: fmt.Sprintf("New%s", structName)},
		Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{
			{Type: &ast.StarExpr{X: &ast.Ident{Name: structName}}},
			{Type: ast.NewIdent("error")},
		}}},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.ExprStmt{X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "viper"},
					Sel: &ast.Ident{Name: "SetConfigFile"},
				},
				Args: []ast.Expr{
					&ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%s\"", configFilePath)},
				},
			}},
			&ast.AssignStmt{
				Lhs: []ast.Expr{&ast.Ident{Name: "err"}},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{&ast.CallExpr{Fun: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "viper"},
					Sel: &ast.Ident{Name: "ReadInConfig"},
				}}},
			},
			errorHandlingStmt,
			&ast.AssignStmt{
				Lhs: []ast.Expr{&ast.Ident{Name: "config"}},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{
					&ast.UnaryExpr{
						Op: token.AND,
						X: &ast.CompositeLit{
							Type: &ast.Ident{Name: structName},
						},
					},
				},
			},
			&ast.AssignStmt{
				Lhs: []ast.Expr{&ast.Ident{Name: "err"}},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: "viper"},
						Sel: &ast.Ident{Name: "Unmarshal"},
					},
					Args: []ast.Expr{
						&ast.Ident{Name: "config"},
					},
				}},
			},
			errorHandlingStmt,
			&ast.ReturnStmt{Results: []ast.Expr{
				&ast.Ident{Name: "config"},
				ast.NewIdent("nil"),
			}},
		}},
	}
	file.Decls = append(file.Decls, funcDefinition)
	file.Name = &ast.Ident{Name: "config"}
	astutil.AddImport(fset, file, "github.com/spf13/viper")
	var code bytes.Buffer
	err = printer.Fprint(&code, fset, file)
	if err != nil {
		return fmt.Errorf("failed to print ast code: %w", err)
	}
	formattedCode, err := format.Source(code.Bytes())
	if err != nil {
		return fmt.Errorf("failed to format code: %w", err)
	}
	err = os.WriteFile(fmt.Sprintf("pkg/infra/config/%s.go", name), formattedCode, 0644)
	if err != nil {
		return fmt.Errorf("failed to write to source file: %w", err)
	}
	return nil
}

func init() {
	configCmd.AddCommand(createCmd)
}
