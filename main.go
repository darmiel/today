package main

import (
	"errors"
	"fmt"
	"github.com/apognu/gocal"
	"github.com/urfave/cli/v2"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	DefaultFormatName = "default"
)

type FormatContext struct {
	event     *gocal.Event
	isCurrent bool
}

type (
	FormatFun     func(ctx *FormatContext) ([]string, error)
	FormatInitFun func() (FormatFun, error)
)

var (
	now        = time.Now()
	formatters = map[string]FormatInitFun{
		DefaultFormatName: createDefaultFormatter,
		"simple":          createSimpleFormatter,
	}
)

const (
	CategoryInput             = "INPUT"
	CategoryCalendarSelection = "CALENDAR"
	CategoryOutput            = "OUTPUT"
	CategoryRALF              = "RALF"
)

func main() {
	app := &cli.App{
		Name:    "today",
		Usage:   "iCal CLI Viewer",
		Version: "1.3.0",
		Authors: []*cli.Author{
			{
				Name:  "darmiel",
				Email: "asdf@qwer.tz",
			},
		},
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:     "path",
				Usage:    "Path of the iCal file",
				Aliases:  []string{"p"},
				EnvVars:  []string{"ICAL_PATH"},
				Category: CategoryInput,
			},
			&cli.PathFlag{
				Name:     "ralf",
				Usage:    "Path of a RALF model",
				EnvVars:  []string{"RALF_DEFINITION"},
				Category: CategoryInput,
			},
			&cli.BoolFlag{
				Name:     "now",
				Usage:    "Show only active events",
				Category: CategoryCalendarSelection,
			},
			// time-start marks the start of the current day (at 00:00:00)
			&cli.TimestampFlag{
				Name:     "time-start",
				Usage:    "Set the start time to show events",
				Value:    cli.NewTimestamp(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)),
				Layout:   "02.01.2006 15:04:05",
				Category: CategoryCalendarSelection,
			},
			// time-end marks the end of the current day (at 23:59:59)
			&cli.TimestampFlag{
				Name:     "time-end",
				Usage:    "Set the end time to show events",
				Value:    cli.NewTimestamp(time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.Local)),
				Layout:   "02.01.2006 15:04:05",
				Category: CategoryCalendarSelection,
			},
			&cli.BoolFlag{
				Name:     "local",
				Usage:    "Convert timestamps to local timezone",
				Value:    true,
				Category: CategoryCalendarSelection,
			},
			// -f specifies formatter
			&cli.StringFlag{
				Name:     "format",
				Usage:    "Formatter for output",
				Aliases:  []string{"f"},
				Category: CategoryOutput,
			},
			&cli.StringFlag{
				Name:     "template",
				Usage:    "Custom formatter",
				Category: CategoryOutput,
			},
			&cli.StringFlag{
				Name:     "join-words",
				Usage:    "Character for joining strings",
				Value:    " ",
				Category: CategoryOutput,
			},
			&cli.StringFlag{
				Name:     "join-lines",
				Usage:    "Character for joining lines",
				Value:    "\n",
				Category: CategoryOutput,
			},
			&cli.BoolFlag{
				Name:     "verbose",
				Usage:    "Verbose output",
				Category: CategoryOutput,
			},
			&cli.BoolFlag{
				Name:     "ralf-verbose",
				Usage:    "Verbose output for RALF flows",
				Category: CategoryRALF,
			},
			&cli.BoolFlag{
				Name:     "ralf-debug",
				Usage:    "Enable RALF debug messages",
				Category: CategoryRALF,
				Value:    true,
			},
			&cli.PathFlag{
				Name:      "ralf-cache",
				Category:  CategoryRALF,
				Usage:     "RALF cache directory",
				EnvVars:   []string{"RALF_CACHE"},
				TakesFile: false,
			},
			&cli.BoolFlag{
				Name:     "list-formats",
				Usage:    "List available formats",
				Aliases:  []string{"L"},
				Category: CategoryOutput,
			},
		},
		Action: func(context *cli.Context) error {
			var (
				// INPUT
				flagPath     = context.Path("path")
				flagRALFPath = context.Path("ralf")
				// CALENDAR
				flagCurrentOnly = context.Bool("now")
				flagStart       = context.Timestamp("time-start")
				flagEnd         = context.Timestamp("time-end")
				flagLocal       = context.Bool("local")
				// OUTPUT
				flagFormatterName = context.String("format")
				flagVerbose       = context.Bool("verbose")
				flagTemplate      = context.String("template")
				flagListFormats   = context.Bool("list-formats")
				// RALF
				flagRALFVerbose = context.Bool("ralf-verbose")
				flagRALFDebug   = context.Bool("ralf-debug")
				flagRALFCache   = context.Path("ralf-cache")
			)

			if flagListFormats {
				fmt.Println("Available formats:", strings.Join(keys(formatters), ", "))
				return nil
			}

			flagStart = ref(time.Date(2023, 05, 04, 0, 0, 0, 0, time.Local))
			flagEnd = ref(time.Date(2023, 05, 04, 23, 0, 0, 0, time.Local))

			// check if formatter exists
			var formatInitFun FormatInitFun
			if flagTemplate != "" {
				if flagFormatterName != "" {
					return errors.New("cannot combine --template and --format")
				}
				formatInitFun = createTemplateFormatter(flagTemplate)
			} else if flagFormatterName != "" {
				var ok bool
				if formatInitFun, ok = formatters[flagFormatterName]; !ok {
					return fmt.Errorf("cannot find formatter: %s", flagFormatterName)
				}
			} else {
				formatInitFun = formatters[DefaultFormatName]
			}

			var reader io.Reader

			// If RALF engine used, modify calendar
			if flagRALFPath != "" {
				r, err := getRALFReader(
					flagPath,
					flagRALFPath,
					flagRALFDebug,
					flagRALFVerbose,
					flagVerbose,
					flagRALFCache,
				)
				if err != nil {
					return err
				}
				if flagVerbose {
					log.Println("Using RALF-engine for calendar modification")
				}
				reader = r
			} else if flagPath != "" {
				// otherwise use "normal" file
				f, err := os.Open(flagPath)
				if err != nil {
					return err
				}
				defer f.Close()
				if flagVerbose {
					fmt.Println("Using normal file open for calendar reading")
				}
				reader = f
			} else {
				return errors.New("you need to specify a path of the iCal file or use the RALF module")
			}

			if flagVerbose {
				fmt.Println("Start:", formatTime(flagStart))
				fmt.Println("End:", formatTime(flagEnd))
			}

			calParser := gocal.NewParser(reader)
			calParser.Start, calParser.End = flagStart, flagEnd
			if err := calParser.Parse(); err != nil {
				panic(err)
			}

			if flagLocal {
				for i, e := range calParser.Events {
					if e.Start != nil {
						e.Start = ref(e.Start.Local())
					}
					if e.End != nil {
						e.End = ref(e.End.Local())
					}
					calParser.Events[i] = e
				}
			}

			// create formatter from init
			formatter, err := formatInitFun()
			if err != nil {
				return err
			}

			type eventPrompt struct {
				event  *gocal.Event
				prompt []string
			}
			var eventPrompts []*eventPrompt

			for _, e := range calParser.Events {
				if e.Start == nil || e.End == nil {
					continue
				}
				isCurrent := now.After(*e.Start) && now.Before(*e.End)
				if flagCurrentOnly && !isCurrent {
					continue
				}
				eventContext := &FormatContext{
					event:     &e,
					isCurrent: isCurrent,
				}
				prompt, err := formatter(eventContext)
				if err != nil {
					return err
				}
				eventPrompts = append(eventPrompts, &eventPrompt{
					event:  &e,
					prompt: prompt,
				})
			}

			// sort by starting time
			sort.Slice(eventPrompts, func(i, j int) bool {
				return eventPrompts[i].event.Start.Before(*eventPrompts[j].event.End)
			})

			var prompts []string
			for _, e := range eventPrompts {
				prompts = append(prompts, strings.Join(e.prompt, context.String("join-words")))
			}
			fmt.Println(strings.Join(prompts, context.String("join-lines")))
			return nil
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatalln("Cannot run app:", err)
		return
	}
}
