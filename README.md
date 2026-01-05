# go-reorder

![go-reorder](reorder.png)

A Go code reordering tool that organizes declarations according to project conventions.

## Overview

`go-reorder` analyzes and reorders Go source code declarations to follow a consistent structure:

1. Package-level comments
2. Imports
3. main() (if present)
4. Exported constants (alphabetically)
5. Exported enums (type + iota block pairs, alphabetically)
6. Exported variables (alphabetically)
7. Exported types (alphabetically)
   - Type definition
   - Constructors (alphabetically)
   - Exported methods (alphabetically)
   - Unexported methods (alphabetically)
8. Exported functions (alphabetically)
9. Unexported constants (alphabetically)
10. Unexported enums (alphabetically)
11. Unexported variables (alphabetically)
12. Unexported types (alphabetically, with same structure as exported)
13. Unexported functions (alphabetically)

## Features

- Preserves all comments and documentation
- Handles enum types (iota blocks paired with their type definitions)
- Groups types with their constructors and methods
- Merges scattered const/var declarations into organized blocks
- Detects and reports section ordering issues

## Installation

```bash
go get github.com/toejough/go-reorder
```

## Usage

### As a Library

```go
package main

import (
    "fmt"
    "os"

    "github.com/toejough/go-reorder"
)

func main() {
    // Read source code
    content, err := os.ReadFile("example.go")
    if err != nil {
        panic(err)
    }

    // Reorder declarations
    reordered, err := reorder.Source(string(content))
    if err != nil {
        panic(err)
    }

    // Write back
    err = os.WriteFile("example.go", []byte(reordered), 0644)
    if err != nil {
        panic(err)
    }
}
```

### Analysis Mode

You can analyze the current section order without modifying files:

```go
order, err := reorder.AnalyzeSectionOrder(sourceCode)
if err != nil {
    panic(err)
}

for _, section := range order.Sections {
    fmt.Printf("%s: position %d (expected %d)\n",
        section.Name, section.Position, section.Expected)
}
```

## API

### `Source(src string) (string, error)`

Reorders declarations in Go source code according to project conventions.

**Parameters:**
- `src`: Go source code as a string

**Returns:**
- Reordered source code
- Error if parsing or processing fails

### `File(file *dst.File) error`

Reorders declarations in a `dst.File` AST node in place.

**Parameters:**
- `file`: Parsed AST file from `github.com/dave/dst`

**Returns:**
- Error if reordering fails (currently always returns nil)

### `AnalyzeSectionOrder(src string) (*SectionOrder, error)`

Analyzes the current declaration order without modifying the source.

**Parameters:**
- `src`: Go source code as a string

**Returns:**
- `SectionOrder` showing which sections are present and their positions
- Error if parsing fails

## Example Integration

This tool was originally developed as part of [imptest](https://github.com/toejough/imptest) and can be integrated into build tools like [Mage](https://magefile.org/):

```go
import "github.com/toejough/go-reorder"

func ReorderDecls() error {
    files, _ := filepath.Glob("**/*.go")

    for _, file := range files {
        content, _ := os.ReadFile(file)
        reordered, err := reorder.Source(string(content))
        if err != nil {
            return err
        }

        if string(content) != reordered {
            os.WriteFile(file, []byte(reordered), 0644)
            fmt.Printf("Reordered: %s\n", file)
        }
    }

    return nil
}
```

## License

See LICENSE file for details.

## Credits

Originally developed as part of the [imptest](https://github.com/toejough/imptest) project.
