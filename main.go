package main

import (
	"flag"
	"fmt"
	"github.com/apognu/gocal"
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

const DefaultFormatName = "default"

type FormatContext struct {
	event     *gocal.Event
	isCurrent bool
}

type FormatFun func(ctx *FormatContext) []string

var (
	now        = time.Now()
	formatters = map[string]FormatFun{
		DefaultFormatName: formatDefault,
		"simple":          formatSimple,
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
						Name:     "path",
						Usage:    "Path of the iCal file",
						Required: true,
						Aliases:  []string{"p"},
						EnvVars:  []string{"ICAL_PATH"},
					},
					&cli.BoolFlag{
						Name:  "now",
						Usage: "Show only active events",
					},
					// -f specifies formatter
					&cli.StringFlag{
						Name:    "format",
						Usage:   "Formatter for output",
						Value:   DefaultFormatName,
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
				},
				Action: func(context *cli.Context) error {
					var (
						flagCurrentOnly   = context.Bool("now")
						flagFormatterName = context.String("format")
						flagPath          = context.Path("path")
					)

					// check if formatter exists
					var (
						formatter FormatFun
						ok        bool
					)
					if formatter, ok = formatters[flagFormatterName]; !ok {
						return fmt.Errorf("cannot find formatter: %s", flagFormatterName)
					}

					// nowStart marks the start of the current day (at 00:00:00)
					nowStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

					// nowEnd marks the end of the current day (at 23:59:59)
					nowEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.UTC)

					f, err := os.Open(flagPath)
					if err != nil {
						panic(err)
					}
					defer f.Close()

					calParser := gocal.NewParser(f)
					calParser.Start, calParser.End = &nowStart, &nowEnd
					if err := calParser.Parse(); err != nil {
						panic(err)
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
						eventPrompts = append(eventPrompts, &eventPrompt{
							event:  &e,
							prompt: formatter(eventContext),
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
