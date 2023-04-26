package main

import (
	"errors"
	"flag"
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
	// EnvCacheDir Default: ~/.local/share/today/cache
	EnvCacheDir = "TODAY_CACHE"
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

func init() {
	flag.Parse()
}

func main() {
	app := &cli.App{
		Name:    "today",
		Usage:   "iCal CLI Viewer",
		Version: "1.0.0",
		Authors: []*cli.Author{
			{
				Name:  "darmiel",
				Email: "asdf@qwer.tz",
			},
		},
		UseShortOptionHandling: true,
		Flags:                  []cli.Flag{},
		Commands: []*cli.Command{
			{
				Name:  "show",
				Usage: "Default usage",
				Flags: []cli.Flag{
					&cli.PathFlag{
						Name:    "path",
						Usage:   "Path of the iCal file",
						Aliases: []string{"p"},
						EnvVars: []string{"ICAL_PATH"},
					},
					&cli.PathFlag{
						Name:  "ralf",
						Usage: "Path of a RALF model",
					},
					&cli.BoolFlag{
						Name:  "now",
						Usage: "Show only active events",
					},
					// time-start marks the start of the current day (at 00:00:00)
					&cli.TimestampFlag{
						Name:   "time-start",
						Usage:  "Set the start time to show events",
						Value:  cli.NewTimestamp(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)),
						Layout: "02.01.2006 15:04:05",
					},
					// time-end marks the end of the current day (at 23:59:59)
					&cli.TimestampFlag{
						Name:   "time-end",
						Usage:  "Set the end time to show events",
						Value:  cli.NewTimestamp(time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.UTC)),
						Layout: "02.01.2006 15:04:05",
					},
					// -f specifies formatter
					&cli.StringFlag{
						Name:    "format",
						Usage:   "Formatter for output",
						Aliases: []string{"f"},
					},
					&cli.StringFlag{
						Name:  "join-words",
						Usage: "Character for joining strings",
						Value: " ",
					},
					&cli.StringFlag{
						Name:  "join-lines",
						Usage: "Character for joining lines",
						Value: "\n",
					},
					&cli.BoolFlag{
						Name:    "verbose",
						Usage:   "Verbose output",
						Aliases: []string{"v"},
					},
					&cli.BoolFlag{
						Name:  "ralf-verbose",
						Usage: "Verbose output for RALF flows",
					},
					&cli.StringFlag{
						Name:  "template",
						Usage: "Custom formatter",
					},
				},
				Action: func(context *cli.Context) error {
					var (
						flagCurrentOnly   = context.Bool("now")
						flagFormatterName = context.String("format")
						flagPath          = context.Path("path")
						flagVerbose       = context.Bool("verbose")
						flagRALFVerbose   = context.Bool("ralf-verbose")
						flagRALFPath      = context.Path("ralf")
						flagStart         = context.Timestamp("time-start")
						flagEnd           = context.Timestamp("time-end")
						flagTemplate      = context.String("template")
					)

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
						r, err := getRALFReader(flagPath, flagRALFPath, flagRALFVerbose, flagRALFVerbose, flagVerbose)
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
						panic("You need to specify a path of the iCal file or use the RALF module.")
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
			},
			{
				Name: "list",
				Subcommands: []*cli.Command{
					{
						Name: "format",
						Action: func(context *cli.Context) error {
							var keys []string
							for k := range formatters {
								keys = append(keys, k)
							}
							log.Println("Available formatters:", strings.Join(keys, ", "))
							return nil
						},
					},
				},
			},
			{
				Name:  "ralf",
				Usage: "RALFated commands",
				Action: func(context *cli.Context) error {
					// today ralf -f model.yaml
					panic("To be implemented")
				},
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatalln("Cannot run app:", err)
		return
	}
}
