package cmd

import (
	"embed"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed help/*.txt
var helpFiles embed.FS

type options struct {
	show bool
	read bool
	list bool
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
		Use:           "mob [FILE] [options]",
		Short:         "Markdown Ops Book",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			document, err := resolveDocument(args)
			if err != nil {
				return err
			}

			switch {
			case opts.show:
				return showDocument(document, stdout)
			case opts.list:
				fmt.Fprintf(stdout, "list %s (not implemented)\n", document)
			default:
				return readDocument(document, cmd.InOrStdin(), stdout)
			}
			return nil
		},
	}

	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetHelpFunc(rootHelp)
	root.SetVersionTemplate("{{.Version}}\n")
	root.CompletionOptions.DisableDefaultCmd = true

	root.Flags().BoolVarP(&opts.show, "show", "s", false, "render the document and exit")
	root.Flags().BoolVarP(&opts.read, "read", "r", false, "open the interactive reader (default)")
	root.Flags().BoolVarP(&opts.list, "list", "l", false, "list executable blocks with their nearest heading")
	root.MarkFlagsMutuallyExclusive("show", "read", "list")

	return root
}

func rootHelp(cmd *cobra.Command, _ []string) {
	content, err := helpFiles.ReadFile("help/root.txt")
	if err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		return
	}
	fmt.Fprint(cmd.OutOrStdout(), string(content))
}

func resolveDocument(args []string) (string, error) {
	if len(args) == 1 {
		if err := requireFile(args[0]); err != nil {
			return "", err
		}
		return args[0], nil
	}

	for _, candidate := range []string{"mob.md", "README.md"} {
		if err := requireFile(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("no document found (expected mob.md or README.md)")
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
