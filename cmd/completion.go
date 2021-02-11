package cmd

import (
	"os"

	"github.com/daedaleanai/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:

  $ source <(dbt completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ dbt completion bash > /etc/bash_completion.d/dbt
  # macOS:
  $ dbt completion bash > /usr/local/etc/bash_completion.d/dbt

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ dbt completion zsh > "${fpath[1]}/_dbt"

  # You will need to start a new shell for this setup to take effect.

fish:

  $ dbt completion fish | source

  # To load completions for each session, execute once:
  $ dbt completion fish > ~/.config/fish/completions/dbt.fish
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		}
	},
	Hidden: true,
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
