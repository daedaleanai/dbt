# Installing the completion scrips

## Bash:

  `source < bash/dbt`

  To load completions for each session, execute once:

  Linux:
  `cp basb/dbt /etc/bash_completion.d/dbt`
  
  macOS:
  `cp basb/dbt /usr/local/etc/bash_completion.d/dbt`

## Zsh:

  If shell completion is not already enabled in your environment,
  you will need to enable it.  You can execute the following once:

  `echo "autoload -U compinit; compinit" >> ~/.zshrc`

  To load completions for each session, execute once:

  `cp zsh/_dbt > "${fpath[1]}/_dbt"`

  You will need to start a new shell for this setup to take effect.

## fish:

  `source < fish/dbt.fish`

  To load completions for each session, execute once:

  `cp fish/dbt.fish ~/.config/fish/completions/dbt.fish`
