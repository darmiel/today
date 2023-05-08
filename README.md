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

## Shell Integration

### Starship

Paste the following contents to `~/.config/starship.toml` and modify to your needs.

```toml
[custom.today]
command = 'today show --now -p "<path to .ics>" -f simple --join-lines ", "'
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
