# today

What's in my calendar today? ` 🔵 With Shell Integration ` ` 🔵 With RALF Integration `

<img width="1119" alt="Screenshot 2023-04-20 at 23 47 43" src="https://user-images.githubusercontent.com/71837281/233494732-dbacaf1f-a2fd-4c40-ac0f-3a1c6cd4768b.png">

## Installation

**🍏 Using Brew**
```bash
brew install darmiel/today/today
```

---

**🛠️ Building from source using Go**

```bash
go install github.com/darmiel/today@latest
```

---

**📦 Precompiled binaries**

Can be found [here](https://github.com/darmiel/today/releases/latest)

## Usage

```bash
today [...Options]

# Using .ics file via -p parameter
today -p /tmp/calendar.ics
# NOTE: the path to the calendar can also be specified by the `ICAL_PATH` environment variable

# Using a RALF definition via --ralf parameter
today --ralf /tmp/definition.yaml
# NOTE: the path to the definition can also be specified by the `RALF_DEFINITION` environment variable
```

### Options

```
GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version

CALENDAR

   --local             Convert timestamps to local timezone (default: true)
   --now               Show only active events (default: false)
   --time-end value    Set the end time to show events (default: 2023-05-10 23:59:59 +0200 CEST)
   --time-start value  Set the start time to show events (default: 2023-05-10 00:00:00 +0200 CEST)

INPUT

   --path value, -p value  Path of the iCal file [$ICAL_PATH]
   --ralf value            Path of a RALF model [$RALF_DEFINITION]

OUTPUT

   --first value             Only output first n events (default: 0)
   --format value, -f value  Formatter for output
   --join-lines value        Character for joining lines (default: "\n")
   --join-words value        Character for joining strings (default: " ")
   --last value              Only output last n events (default: 0)
   --list-formats, -L        List available formats (default: false)
   --template value          Custom formatter
   --verbose                 Verbose output (default: false)
   --write-file value        Write iCal to file
   --write-stdout            Write iCal to stdout (default: true)

RALF

   --ralf-cache value  RALF cache directory [$RALF_CACHE]
   --ralf-debug        Enable RALF debug messages (default: true)
   --ralf-verbose      Verbose output for RALF flows (default: false)
```

### today as a proxy

You can use `today` to only modify the calendar using the `RALF` integration.

```bash
$ cat definition.yaml
source: file:///tmp/calendar.ics

flows:
  - do: actions/regex-replace
    with:
      case-sensitive: true
      in: [ "summary" ]
      map:
        - match: "[rl]"
          replace: "w"
        - match: "[RL]"
          replace: "W"
        - match: "n([aeiou])"
          replace: "ny$1"
        - match: "N([aeiouAEIOU])"
          replace: "Ny$1"
        - match: "ove"
          replace: "uv"

$ today --ralf definition.yaml --write-file /tmp/calendar.ics --write-stdout='false'
```

### Formats

You can show a list of available formats using `-L`.

```bash
$ today -L
Available formats: default, simple, raw

$ today -p ... -f simple
Test Event [1h55m19s remaining]
```

If the predefined formats don't fit your needs, you can create a custom format by using the `--template`-Flag.

```bash
$ today -p ... --template '({{ .Relative }}) {{ .Event.Summary }}'
(1h55m19s remaining) Test Event
```

Available template context:

```go
type TemplateContext struct {
    Event       *gocal.Event
    Relative    string
    RelativeRAW string
    Progress    string
    Start       *engine.CtxTime
    End         *engine.CtxTime
    IsCurrent   bool
}

```

## Shell Integration

### Starship

Paste the following contents to `~/.config/starship.toml` and modify to your needs.

```toml
[custom.today]
command = 'today --now -p "<path to .ics>" -f simple --join-lines ", "'
when = true
style = 'bold green'
symbol = '📆 '
```

## Filtering / Modifying the Calendar

For filtering or modifying the calendar, 
*today* has a native [RALF](https://github.com/ralf-life/engine) integration.

You can create a YAML-file with a *RALF-Definition* and pass the file name using `--ralf`:

```bash
$ cat definition.yaml
---
flows:
  - if: 'Event.Summary() contains "Numerik"'
    then:
      - do: filters/filter-out
    else:
      - do: actions/regex-replace
        with:
          match: "AdA"
          replace: "AbbA"
          in: [ "summary" ]
...
```
> **Note**: You can read the blog post about the RALF-Syntax [here](https://the.ralf.life/gh-ralf-speck)

```bash
# before:
$ today show -p ... -f simple
> Numerik from 08:00 to 11:45 [ 2h45m21s / 3h45m     ] (59m38s remaining)
  AdA [3h15m0s] from 12:30 to 15:45 (in 3h44m38s)

# after
$ today show -p ... -f simple --ralf definition.yaml
  AbbA [3h15m0s] from 12:30 to 15:45 (in 3h44m38s)
```
