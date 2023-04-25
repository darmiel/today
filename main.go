package main

import (
	"errors"
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
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	DefaultFormatName = "default"
	// ~/.local/share/today/cache
	EnvCacheDir = "TODAY_CACHE"
)

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

var ErrSourceMissing = errors.New("cannot find source")
var ErrInvalidSource = errors.New("source protocol not supported")
var ErrCacheNotDir = errors.New("cache must be a directory")

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
					} else if flagPath != "" {
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
					} else {
						panic("You need to specify a path of the iCal file or use the RALF module.")
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

	var r io.Reader

	// iCal source was specified by flag
	if iCalPath != "" {
		// parse calendar
		f, err := os.Open(iCalPath)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		r = f
	} else if profile.Source != "" {
		// iCal source was not specify, get from profile (`source`)
		u, err := url.Parse(profile.Source)
		if err != nil {
			return nil, err
		}
		switch u.Scheme {
		case "http", "https":
			// load iCal file via http

			// temporary directory
			tempDir, ok := os.LookupEnv(EnvCacheDir)
			if !ok {
				tempDir = "~/.local/share/today/cache"
			}
			if stat, err := os.Stat(tempDir); os.IsNotExist(err) {
				if verbose {
					fmt.Println("creating cache directory at", tempDir)
				}
				if err = os.MkdirAll(tempDir, os.ModePerm); err != nil {
					return nil, err
				}
			} else if stat != nil && !stat.IsDir() {
				return nil, ErrCacheNotDir
			}

			fileName := filepath.Join(tempDir, filepath.Base(ralfPath)+".cached.ics")
			var duration time.Duration
			if int64(profile.CacheDuration) > 0 {
				duration = time.Duration(profile.CacheDuration)
			} else {
				duration = 5 * time.Minute
			}
			if stat, err := os.Stat(fileName); os.IsNotExist(err) ||
				(stat != nil && time.Now().After(stat.ModTime().Add(duration))) {
				// (re-)download
				fmt.Println("needing to re-download file")
				resp, err := http.Get(profile.Source)
				if err != nil {
					return nil, err
				}
				defer resp.Body.Close()
				f, err := os.Create(fileName)
				if err != nil {
					return nil, err
				}
				defer f.Close()
				if _, err := io.Copy(f, resp.Body); err != nil {
					return nil, err
				}
			}
			f, err := os.Open(fileName)
			if err != nil {
				return nil, err
			}
			defer f.Close()
			r = f
		case "file":
			// load iCal file from system
			f, err := os.Open(u.Path)
			if err != nil {
				return nil, err
			}
			defer f.Close()
			r = f
		default:
			return nil, ErrInvalidSource
		}
	} else {
		return nil, ErrSourceMissing
	}

	cal, err := ics.ParseCalendar(r)
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
