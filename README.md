# piko

Spin up isolated dev environments in seconds. No port conflicts. No stashing. No context-switching pain.

```
piko env create feature-auth      # worktree + containers + tmux
piko env create feature-payments  # another isolated environment
piko env destroy feature-auth     # teardown everything
```

## Concepts

**Project**: A git repository initialized with `piko init`. Contains multiple environments.

**Environment**: An isolated workspace for a feature or task. Each one gets:

- A git worktree (its own branch and working directory)
- A tmux session (persistent terminal)
- Docker containers with unique ports (if you use docker-compose)
- A data directory for local state

No conflicts between environments. Switch contexts instantly.

## Install

```
go install github.com/gwuah/piko@latest
```

Requires: git, tmux. Docker optional.

## Usage

```bash
piko init                    # initialize project
piko env create my-feature   # create environment
piko env list                # see all environments
piko env destroy my-feature  # remove everything
```

Run `piko --help` for all commands.

## Environment Variables

Available in your scripts and tmux sessions:

```
PIKO_ENV_NAME   # environment name
PIKO_ENV_ID     # unique integer (for port derivation)
PIKO_ROOT       # project root
```

## Coding Agents

With first-class support for coding agents, piko provides a central UI to manage your enviroments & agents.

```bash
piko cc init    # set up hooks in current environment
piko server     # manage all agents at localhost:19876
```

## Configuration

Optional `.piko.yml`:

```yaml
scripts:
  setup: npm install
  run: npm run dev
```

## License

MIT
