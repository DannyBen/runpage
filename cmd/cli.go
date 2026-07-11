package cmd

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed help/*.txt
var helpFiles embed.FS

type options struct {
	show    bool
	read    bool
	compact int
	syntax  bool
	workdir string
}

func Execute(args []string, version string, stdout, stderr io.Writer) error {
	return ExecuteWithIO(args, version, strings.NewReader(""), stdout, stderr)
}

func ExecuteWithIO(args []string, version string, stdin io.Reader, stdout, stderr io.Writer) error {
	root := NewRootCommand(version, stdout, stderr)
	root.SetIn(stdin)
	root.SetArgs(args)
	return root.Execute()
}

func PrintError(err error, stderr io.Writer) {
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
	}
}

func NewRootCommand(version string, stdout, stderr io.Writer) *cobra.Command {
	var opts options

	root := &cobra.Command{
		Use:           "runpage [FILE] [options]",
		Short:         "Interactive command pages for the terminal",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.syntax {
				if len(args) > 0 {
					return fmt.Errorf("--syntax does not accept arguments")
				}
				return printHelpFile(cmd, "help/syntax.txt")
			}
			documentArgs := args
			valueArgs := []string(nil)
			if len(args) > 0 {
				documentArgs = args[:1]
				valueArgs = args[1:]
			}
			document, err := resolveDocument(documentArgs)
			if err != nil {
				return err
			}
			values, err := parseDocumentValues(valueArgs)
			if err != nil {
				return err
			}
			loaded, err := loadDocument(document, values)
			if err != nil {
				return err
			}
			if !opts.read {
				if err := requireDocumentValues(loaded.config, values); err != nil {
					return err
				}
			}

			workdir := ""
			if !opts.read && !opts.show {
				if err := requireDependencies(loaded.config); err != nil {
					return err
				}
				workdir, err = resolveDocumentWorkdir(loaded.config, opts.workdir, document)
				if err != nil {
					return err
				}
			}

			switch {
			case opts.show:
				return showDocument(loaded.markdown, stdout)
			default:
				return readDocument(loaded.markdown, cmd.InOrStdin(), stdout, opts.read, opts.compact, workdir)
			}
		},
	}

	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetHelpFunc(rootHelp)
	root.SetVersionTemplate("{{.Version}}\n")
	root.CompletionOptions.DisableDefaultCmd = true

	root.Flags().BoolVarP(&opts.show, "show", "s", false, "render the document and exit")
	root.Flags().BoolVarP(&opts.read, "read", "r", false, "open the interactive reader without execution")
	root.Flags().CountVarP(&opts.compact, "compact", "c", "hide prose; repeat to also hide executable code")
	root.Flags().BoolVar(&opts.syntax, "syntax", false, "show the Runpage document syntax")
	root.Flags().StringVarP(&opts.workdir, "workdir", "w", "", "directory used to execute code blocks")
	root.MarkFlagsMutuallyExclusive("show", "read", "compact", "syntax")

	return root
}

func resolveWorkdir(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve working directory %s: %w", path, err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("working directory not found: %s", path)
		}
		return "", fmt.Errorf("open working directory %s: %w", path, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("working directory is not a directory: %s", path)
	}
	return absolute, nil
}

func rootHelp(cmd *cobra.Command, _ []string) {
	if err := printHelpFile(cmd, "help/root.txt"); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
	}
}

func printHelpFile(cmd *cobra.Command, path string) error {
	content, err := helpFiles.ReadFile(path)
	if err != nil {
		return err
	}
	fmt.Fprint(cmd.OutOrStdout(), string(content))
	return nil
}

func resolveDocument(args []string) (string, error) {
	if len(args) == 1 {
		if err := requireFile(args[0]); err != nil {
			return "", err
		}
		return args[0], nil
	}

	for _, candidate := range []string{"runpage.md", "README.md"} {
		if err := requireFile(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("no document found (expected runpage.md or README.md)")
}

func requireFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("document not found: %s", path)
		}
		return fmt.Errorf("open document %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("document is a directory: %s", path)
	}
	return nil
}
