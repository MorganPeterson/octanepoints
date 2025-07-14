# octanepoints

A cross-platform CLI tool written in Go to tally rally points at the end of a rally event. Supports both Windows and Linux.

I develop this application on Linux and do some limited testing on Windows.

## Features

* Parse stage result files (CSV) (semicolon separator)
* Calculate points based on configurable scoring rules
* Output standings in CSV
* Easy configuration via config file

## Prerequisites

* Go 1.19 or later; Go 1.24 preferred
* Git
* Sqlite 3

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

This application relys on some manual labor on your part.

1. Go to the rally's summary page after the rally has finished.
2. Find the rally id in the summary page url (ex. 15234)
3. Create a folder/directory named the rally id (ex. 15234) in your `rallies` folder.
4. In that directory create a TOML file named the rally id (ex. 15234.toml)
5. Fill out the TOML file with the information from the summary page as below.

```toml
[rally]
rallyId = 15234
name = "My Rally"
description = "All Rally, All Day."
creator = "Your Name"
damageLevel = "reduced"
numberOfLegs = 3
superRally = true
pacenotesOptions = "Normal Pacenotes"
started = 19
finished = 14
totalDistance = 151.1
carGroups = "Super 2000, Group B"
startAt = "2025-06-24 11:00"
endAt = "2025-07-01 11:00"

```

6. Go to the rally's result page and navigate to the bottom. There will be 2
    links: `Export results.csv (beta)` and `Export results.csv Final Standing`
7. Download both files to the `rallies/[rally id]` directory which also contains
    your TOML file.

Your rallies directory will look something like this:

```bash
rallies
├── 15234
│   ├── 15234_All_table.csv
│   ├── 15234_table.csv
│   └── 15234.toml
```

You are all set now to `create` the rally (which puts the data into the database),
and then run `points` to collect points on the rally.

```bash
./octanepoints -create 12345 // will load rally 12345 data into the database

./octanepoints -points 12345 // will print out results and points of rally 12345
```

Once a rally is "created" and loaded into the database, you will never have to 
create it again. You can run "points" and it will just compute the results from
the data in the database.

## Configuration

The scoring configuration uses TOML format. The config file should be named `config.toml` and placed in the directory you are running the application in. A configuration file
is included in the repo.

```toml
[general]
points = [32, 28, 25, 22, 20, 18, 16, 14, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1]
headers = "pos,driver,points"   // headers for printed out points CSV
descriptionDir = "rallies"     // directory name that contains rally CSV files

[database]
name = "octanepoints.db" // name of the database file
```

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
