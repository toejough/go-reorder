# Configuration Examples

Copy one of these files to `.go-reorder.toml` in your project root and customize as needed.

## Available Configs

### webapp.toml
For web applications. Puts HTTP handlers (exported functions) at the bottom of files for easy discovery. Models and internal helpers come first.

### cli-tool.toml
For CLI tools. Puts `main()` at the top so the entry point is immediately visible. Good for command-line applications using cobra, urfave/cli, or similar.

### minimal.toml
A lenient config that just organizes code without strict section requirements. Uses `mode = "append"` so unmatched code doesn't cause errors. Good for legacy codebases or gradual adoption.

## Usage

```bash
# Copy to your project
cp examples/webapp.toml .go-reorder.toml

# Or generate default config and customize
go-reorder --init
```

## Creating Your Own

Run `go-reorder --list-sections` to see all available section names:

```
imports, main, init
exported_consts, exported_enums, exported_vars, exported_types, exported_funcs
unexported_consts, unexported_enums, unexported_vars, unexported_types, unexported_funcs
uncategorized
```

Omit sections you don't care about ordering. Include `uncategorized` as a catch-all, or use `mode = "append"` in `[behavior]` to handle unmatched code.
