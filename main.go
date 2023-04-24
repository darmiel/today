package main

import (
	"flag"
	"fmt"
	"github.com/apognu/gocal"
	ics "github.com/darmiel/golang-ical"
	"github.com/ralf-life/engine/actions"
	"github.com/ralf-life/engine/engine"
	"github.com/ralf-life/engine/model"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
	"io"
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
					&cli.PathFlag{
						Name:  "ralf",
						Usage: "Path of a RALF model",
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
					&cli.BoolFlag{
						Name:  "verbose",
						Usage: "Verbose output",
					},
				},
				Action: func(context *cli.Context) error {
					var (
						flagCurrentOnly   = context.Bool("now")
						flagFormatterName = context.String("format")
						flagPath          = context.Path("path")
						flagVerbose       = context.Bool("verbose")
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

					var reader io.Reader

					// If RALF engine used, modify calendar
					if ralfPath := context.Path("ralf"); ralfPath != "" {
						if r, err := getRALFReader(flagPath, ralfPath, flagVerbose); err != nil {
							return err
						} else {
							if flagVerbose {
								log.Println("Using RALF-engine for calendar modification")
							}
							reader = r
						}
					} else {
						// otherwise use "normal" file
						if f, err := os.Open(flagPath); err != nil {
							return err
						} else {
							if flagVerbose {
								fmt.Println("Using normal file open for calendar reading")
							}
							reader = f
							defer f.Close()
						}
					}

					calParser := gocal.NewParser(reader)
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

func getRALFReader(iCalPath, ralfPath string, verbose bool) (io.Reader, error) {
	rf, err := os.Open(ralfPath)
	if err != nil {
		return nil, err
	}
	defer rf.Close()

	var profile model.Profile
	dec := yaml.NewDecoder(rf)
	dec.KnownFields(true)
	if err = dec.Decode(&profile); err != nil {
		return nil, err
	}
	cp := engine.ContextFlow{
		Profile:     &profile,
		Context:     make(map[string]interface{}),
		EnableDebug: verbose,
		Verbose:     verbose,
	}

	// parse calendar
	f, err := os.Open(iCalPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	cal, err := ics.ParseCalendar(f)
	if err != nil {
		return nil, err
	}

	// get components from calendar (events) and copy to slice for later modifications
	cc := cal.Components[:]

	// start from behind so we can remove from slice
	for i := len(cc) - 1; i >= 0; i-- {
		event, ok := cc[i].(*ics.VEvent)
		if !ok {
			continue
		}
		var fact actions.ActionMessage
		if fact, err = cp.RunAllFlows(event, profile.Flows); err != nil {
			if err == engine.ErrExited {
				if verbose {
					log.Println("[RALF] flows exited because of a return statement.")
				}
			} else {
				return nil, err
			}
		}
		switch fact.(type) {
		case actions.FilterOutMessage:
			cc = append(cc[:i], cc[i+1:]...) // remove event from components
		}
	}
	cal.Components = cc
	return strings.NewReader(cal.Serialize()), nil
}
