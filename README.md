# octanepoints

A cross-platform CLI tool written in Go to tally rally points at the end of a rally event. Supports both Windows and Linux.

## Features

* Parse stage result files (CSV) (semicolon separator)
* Calculate points based on configurable scoring rules
* Output standings in CSV
* Easy configuration via config file

## Prerequisites

* Go 1.19 or later; Go 1.24 preferred
* Git

## Installation

```bash
# Clone the repository
git clone https://git.sr.ht/~nullevoid/octanepoints
cd octanepoints
```

## Building

### On Linux

```bash
go build -o octanepoints .
```

### On Windows

```bash
set GOOS=windows
set GOARCH=amd64
go build -o octanepoints.exe .
```

#### Cross-compiling from any OS

```bash
# Linux amd64
go build -o octanepoints-linux .

# Windows amd64
go build -o octanepoints.exe .
```

## Usage

```bash
./octanepoints <rallyid>
```

## Configuration

The scoring configuration uses TOML format. The config file should be named `config.toml` and placed in the directory you are running the application in. A configuration file
is included in the repo.

```toml
cg = 7
url = "https://rallysimfans.hu/rbr/csv_export_results.php?rally_id=%d&cg=%d"
points = [32, 28, 25, 22, 20, 18, 16, 14, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1]
headers = "pos,driver,points"
```

## Examples

Download a CSV and output standings as CSV:

```bash
./octanepoints 84859
```

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
