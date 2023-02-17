package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"golang.org/x/tools/go/ast/astutil"
	"os"
	"os/exec"
	"strings"
	"template/config"
)

const (
	applicationTypeApi = "api"
	applicationTypeCli = "cli"
)

var forceCreate bool
var applicationType string

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "A command to initialize new project",
	Long:  `This command allows you to create new project, given the project's name`,
	Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		applicationName := args[0]
		err := createDirectory(applicationName, forceCreate) // todo delegate project structure creation to separate unit
		if err != nil {
			return err
		}
		defer func() {
			if err != nil {
				removalErr := os.RemoveAll(applicationName)
				if removalErr != nil {
					err = fmt.Errorf("%w\nfailed to remove created directory: %s", err, removalErr)
				}
			}
		}()
		err = os.Chdir(applicationName)
		if err != nil {
			return err
		}
		err = initializeProject(applicationName)
		if err != nil {
			return err
		}
		err = initializeEntrypoint(applicationType)
		if err != nil {
			return err
		}
		err = initializeConfig(applicationName, applicationType, config.ConfigLibraryName)
		if err != nil {
			return err
		}
		err = initializeGeneratorData()
		if err != nil {
			return err
		}
		return nil
	},
}

func createDirectory(applicationName string, force bool) error {
	if force {
		err := os.RemoveAll(applicationName)
		if err != nil {
			return fmt.Errorf("failed to remove existing directory")
		}
	}
	err := os.Mkdir(applicationName, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create dir for project: %w", err)
	}
	return nil
}

func initializeProject(projectName string) error {
	cmd := exec.Command("go", "mod", "init", projectName)
	err := cmd.Run()
	if err != nil {
		return err
	}
	cmd = exec.Command("go", "mod", "tidy")
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

// todo implement api and use gin/echo for it. Create dir for middleware
func initializeEntrypoint(appType string) error {
	firstEntrypointDirector := fmt.Sprintf("cmd/%s", appType)
	var contents string
	if appType == "api" {
		cmd := exec.Command("go", "get", "-u", "github.com/gin-gonic/gin")
		err := cmd.Run()
		if err != nil {
			return err
		}
		contents = `package main

import (
	"fmt"
  "net/http"

  "github.com/gin-gonic/gin"
)

func main() {
	fmt.Printf("Application launched!")
  r := gin.Default()
  r.GET("/", func(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
      "hello": "Hello world!",
    })
  })
  r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}`
	} else {
		contents = `package main

import (
	"fmt"
)

func main() {
	fmt.Printf("Hello world!")
}`
	}
	err := os.MkdirAll(firstEntrypointDirector, os.ModePerm)
	if err != nil {
		return err
	}
	main, err := os.Create(fmt.Sprintf("%s/%s", firstEntrypointDirector, "main.go")) // todo delegate creating entrypoint to separate module, seeded with init data
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(main, contents)
	if err != nil {
		return err
	}
	return nil
}

// todo after init, possibility to add new config keys and config files with separate structures
func initializeConfig(appName, appType, configLibName string) error {
	// create config directory structure
	const (
		configFileDirectory     = "config"
		configSourceDirectory   = "pkg/infra/config" // todo delegate creating this structure to separate module
		initialConfigFileName   = "application.yaml"
		initialConfigSourceName = "application.go"
	)
	initialConfigurationFileContents := fmt.Sprintf("ApplicationName: %s", appName)
	initialConfigurationSourceContents := `package config

import (
	"github.com/spf13/viper"
)

type ApplicationConfig struct {
	ApplicationName string ` + "`yaml:\"ApplicationName\"`" + `
}

func NewApplicationConfig() (*ApplicationConfig, error) {
	viper.SetConfigFile("config/application.yaml")
	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}
	config := &ApplicationConfig{}
	err = viper.Unmarshal(config)
	if err != nil {
		return nil, err
	}
	return config, nil
}`
	cmd := exec.Command("go", "get", configLibName)
	err := cmd.Run()
	if err != nil {
		return err
	}
	err = os.Mkdir(configFileDirectory, os.ModePerm)
	if err != nil {
		return err
	}
	err = os.MkdirAll(configSourceDirectory, os.ModePerm)
	if err != nil {
		return err
	}
	yaml, err := os.Create(fmt.Sprintf("%s/%s", configFileDirectory, initialConfigFileName))
	if err != nil {
		return err
	}
	defer yaml.Close()
	_, err = fmt.Fprintln(yaml, initialConfigurationFileContents)
	if err != nil {
		return err
	}
	source, err := os.Create(fmt.Sprintf("%s/%s", configSourceDirectory, initialConfigSourceName))
	if err != nil {
		return err
	}
	defer source.Close()
	_, err = fmt.Fprintln(source, initialConfigurationSourceContents)
	if err != nil {
		return err
	}

	// alter main.go code
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, fmt.Sprintf("cmd/%s/main.go", appType), nil, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return err
	}
	funcName := "main"
	for _, decl := range node.Decls {
		if f, ok := decl.(*ast.FuncDecl); ok && f.Name.Name == funcName {
			configAssignmentStatement := &ast.AssignStmt{
				Lhs: []ast.Expr{
					&ast.Ident{Name: "applicationConfig"},
					&ast.Ident{Name: "err"},
				},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{
					&ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   &ast.Ident{Name: "config"},
							Sel: &ast.Ident{Name: "NewApplicationConfig"},
						},
					},
				},
			}
			errorCheckStatement := &ast.IfStmt{
				Cond: &ast.BinaryExpr{
					X:  &ast.Ident{Name: "err"},
					Op: token.NEQ,
					Y:  &ast.Ident{Name: "nil"},
				},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.ExprStmt{
							X: &ast.CallExpr{
								Fun: &ast.SelectorExpr{
									X:   &ast.Ident{Name: "log"},
									Sel: &ast.Ident{Name: "Fatalf"},
								},
								Args: []ast.Expr{
									&ast.BasicLit{
										Kind:  token.STRING,
										Value: `"Failed to read configuration file: %v"`,
									},
									&ast.Ident{Name: "err"},
								},
							},
						},
					},
				},
			}

			// find and alter print statement
			for _, statement := range f.Body.List {
				if expressionStatement, ok := statement.(*ast.ExprStmt); ok {
					if callExpression, ok := expressionStatement.X.(*ast.CallExpr); ok {
						if selectorExpression, ok := callExpression.Fun.(*ast.SelectorExpr); ok {
							if ident, ok := selectorExpression.X.(*ast.Ident); ok {
								if ident.Name == "fmt" && selectorExpression.Sel.Name == "Printf" {
									previousMessage := strings.Trim(callExpression.Args[0].(*ast.BasicLit).Value, "\"")
									callExpression.Args = []ast.Expr{
										&ast.BasicLit{
											Kind:  token.STRING,
											Value: fmt.Sprintf("\"%s %s\"", previousMessage, "Welcome to %s!\\n"),
										},
										&ast.SelectorExpr{
											X:   &ast.Ident{Name: "applicationConfig"},
											Sel: &ast.Ident{Name: "ApplicationName"},
										},
									}
									break
								}
							}
						}
					}
				}
			}

			f.Body.List = append([]ast.Stmt{configAssignmentStatement, errorCheckStatement}, f.Body.List...)
			astutil.AddImport(fset, node, "log")
			astutil.AddImport(fset, node, fmt.Sprintf("%s/%s", appName, configSourceDirectory))
			ast.SortImports(fset, node)
		}
	}

	main, err := os.Create(fmt.Sprintf("cmd/%s/main.go", appType))
	if err != nil {
		return err
	}
	defer main.Close()
	err = printer.Fprint(main, fset, node)
	if err != nil {
		return err
	}
	return nil
}

func initializeGeneratorData() error {
	err := os.Mkdir("generator", os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func init() {
	initCmd.Flags().BoolVarP(&forceCreate, "forceCreate", "f", false, "This flag makes it possible to create new, clean project, even if directory with the same name already exists")
	initCmd.Flags().StringVarP(&applicationType, "applicationType", "t", applicationTypeCli, "Decide whether first app's entrypoint should be cli or api") // todo implement api
	rootCmd.AddCommand(initCmd)
}
