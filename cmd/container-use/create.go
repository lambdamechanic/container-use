package main

import (
    "fmt"
    "os"
    "strings"

    "dagger.io/dagger"
    "github.com/dagger/container-use/repository"
    "github.com/spf13/cobra"
)

var (
    fromGitRef   string
    explanation  string
)

var createCmd = &cobra.Command{
    Use:   "create <title>",
    Short: "Create a new environment",
    Long: `Create a new development environment using your current container-use configuration.
The environment starts from your repository at the specified git reference (default: HEAD),
applies the configured base image and setup/install commands, and tracks work in an isolated branch.`,
    Args: cobra.MinimumNArgs(1),
    Example: `# Create a new environment from HEAD
container-use create "Experiment: refactor API"

# Create from a specific branch
container-use create --from-git-ref main "Smoke test from main"

# Add an explanation that appears in the log
container-use create -e "Initial scaffold for feature work" "Prototype: auth flow"`,
    RunE: func(cmd *cobra.Command, args []string) error {
        ctx := cmd.Context()

        // Open the repository (must be in a git repo)
        repo, err := repository.Open(ctx, ".")
        if err != nil {
            return err
        }

        // Connect to Dagger
        dag, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
        if err != nil {
            if isDockerDaemonError(err) {
                handleDockerDaemonError()
            }
            return fmt.Errorf("failed to connect to dagger: %w", err)
        }
        defer dag.Close()

        // Build title from remaining args to allow unquoted multi-word titles
        title := strings.Join(args, " ")

        env, err := repo.Create(ctx, dag, title, explanation, fromGitRef)
        if err != nil {
            return fmt.Errorf("failed to create environment: %w", err)
        }

        fmt.Printf("Environment created: %s\n", env.ID)
        if env.State != nil && env.State.Title != "" {
            fmt.Printf("Title: %s\n", env.State.Title)
        }

        // Warn if repository has uncommitted changes that were not included
        if dirty, status, err := repo.IsDirty(ctx); err == nil && dirty {
            fmt.Printf(`\nCRITICAL: Uncommitted changes in %s are NOT included in this environment.\nThe environment was created from the last committed state only.\n\nUncommitted changes detected:\n%s\n\nTo include these changes, commit them first using git commands outside the environment.\n`, repo.SourcePath(), status)
        }

        fmt.Printf("\nNext steps:\n")
        fmt.Printf("- Inspect log:     container-use log %s\n", env.ID)
        fmt.Printf("- Check out files: container-use checkout %s\n", env.ID)
        fmt.Printf("- Open terminal:   container-use terminal %s\n", env.ID)

        return nil
    },
}

func init() {
    createCmd.Flags().StringVar(&fromGitRef, "from-git-ref", "HEAD", "Git reference to create the environment from (e.g., HEAD, main, feature-branch, SHA)")
    createCmd.Flags().StringVarP(&explanation, "explanation", "e", "", "Explanation for why the environment was created (appears in the development log)")
    rootCmd.AddCommand(createCmd)
}

