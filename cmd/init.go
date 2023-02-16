package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"os"
	"os/exec"
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
		err = initializeEntrypoint()
		if err != nil {
			return err
		}
		err = initializeConfig(applicationName, config.ConfigLibraryName)
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
func initializeEntrypoint() error {
	const entrypointDirectoryName = "cmd"
	const contents = `package main

import (
	"fmt"
)

func main() {
	fmt.Printf("Hello world!")
}`
	err := os.Mkdir(entrypointDirectoryName, os.ModePerm)
	if err != nil {
		return err
	}
	main, err := os.Create(fmt.Sprintf("%s/%s", entrypointDirectoryName, "main.go")) // todo delegate creating first entrypoint to separate module, seeded with init data
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
func initializeConfig(appName, configLibName string) error {
	// create config directory structure
	const (
		configFileDirectory     = "config"
		configSourceDirectory   = "pkg/infra/config" // todo delegate creating this structure to separate module
		initialConfigFileName   = "application.yaml"
		initialConfigSourceName = "application.go"
	)
	initialConfigurationFileContents := fmt.Sprintf("application_name: %s", appName)
	initialConfigurationSourceContents := `package config

import (
	"github.com/spf13/viper"
)

type ApplicationConfig struct {
	applicationName string ` + "`yaml:\"application_name\"`" + `
}

func NewApplicationConfig() (*ApplicationConfig, error) {
	viper.SetConfigType("yaml")
	viper.SetConfigName("config/application")
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
}

func (ac *ApplicationConfig) ApplicationName() string {
	return ac.applicationName
}`
	//mainSourceFileConfigAddition := `applicationConfig, err := config.NewApplicationConfig()
	//if err != nil {
	//	log.Fatalln(err)
	//}
	//applicationName := applicationConfig.ApplicationName()`

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
	node, err := parser.ParseFile(fset, "cmd/main.go", nil, parser.AllErrors)
	if err != nil {
		return err
	}
	funcName := "main"
	for _, decl := range node.Decls {
		if f, ok := decl.(*ast.FuncDecl); ok && f.Name.Name == funcName {
			// Add a new statement at the beginning of the function body.
			newStmt := &ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: "fmt"},
						Sel: &ast.Ident{Name: "Println"},
					},
					Args: []ast.Expr{
						&ast.BasicLit{
							Kind:  token.STRING,
							Value: `"Hello, world!"`,
						},
					},
				},
			}
			f.Body.List = append([]ast.Stmt{newStmt}, f.Body.List...)

			// Type-check the modified AST to ensure it's still valid Go code.
			conf := types.Config{Importer: importer.Default()}
			info := types.Info{Types: make(map[ast.Expr]types.TypeAndValue)}
			_, err := conf.Check("cmd/main.go", fset, []*ast.File{node}, &info)
			if err != nil {
				return err
			}

			// Print the modified source code.
			//if err := printer.Fprint(os.Stdout, fset, node); err != nil {
			//	return err
			//}
		}
	}

	main, err := os.Create("cmd/main.go")
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

func init() {
	initCmd.Flags().BoolVarP(&forceCreate, "forceCreate", "f", false, "This flag makes it possible to create new, clean project, even if directory with the same name already exists")
	initCmd.Flags().StringVarP(&applicationType, "applicationType", "t", applicationTypeCli, "Decide whether first app's entrypoint should be cli or api") // todo implement api
	rootCmd.AddCommand(initCmd)
}
