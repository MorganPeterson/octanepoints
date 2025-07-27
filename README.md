# octanepoints

A cross-platform CLI tool written in Go to tally rally points for RBRRSF at the 
end of a rally event.

Supports both Windows and Linux.

I develop this application on Linux and do some limited testing on Windows.

This application is under heavy development and could change at any time. For
better or for worse. Use at your own risk.

## Features

* Download and parse stage and overall result files (CSV) (semicolon separator)
* Calculate points based on configurable scoring rules
* Output standings in Markdown
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
3. Run the `grab` function with the rally id. `./octanepoints -grab 15234`. This
will populate the directory with the necessary files.
4. In that directory, open the TOML file named the rally id (ex. 15234.toml)
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
./octanepoints -create 15234 // will load rally 15234 data into the database

./octanepoints -report 15234 // will generate a report for rally 15234

./octanepoints -summary // will generate general stats for the whole season

./octanepoints -driver 15234 // will generate individual driver stats for rally 15234

./octanepoints -class 15234 // will generate a class report for rally 15234
```

Once a rally is "created" and loaded into the database, you will never have to 
create it again. You can run "points" and it will just compute the results from
the data in the database.

## Configuration

The scoring configuration uses TOML format. The config file should be named 
`config.toml` and placed in the directory you are running the application in. A 
configuration file example is included in the repo named `sample_config.toml`.
You can rename that file to `config.toml` and change fields where necessary.

```toml
[general]
points = [32, 28, 25, 22, 20, 18, 16, 14, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1]
classPoints = [32, 28, 25, 22, 20, 18, 16, 14, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1]
classesType = "driver" # Options: "car", "driver"
descriptionDir = "rallies"
reportDir = "rally_reports"

[database]
name = "rally_data.db"

# classes are optional but some reports (class specific) will not work
[[classes]]
name = "Gold"
description = "Gold Class Drivers"
categories = ["Group B", "Group 4"] # used if classType == "car"
drivers = ["Fred Fast", "Chris Champion", "Frank Ferrari"] # used if classType == "driver"

[[classes]]
name = "Silver"
description = "Silver Class Drivers"
categories = ["Group R4", "Group N4"] # used if classType == "car"
drivers = ["Amy Amatuer", "Niel Young", "Stever Silver"] # used if classType == "driver"
```

## Roadmap

    * Export to PDF
    * Automate more of the directory and file CRUD
    * Add the ability to update a rally description TOML file in program (perhaps automatically)
 
## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
