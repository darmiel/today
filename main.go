package main

import (
	"fmt"
	"github.com/apognu/gocal"
	"github.com/fatih/color"
	"math"
	"os"
	"sort"
	"strings"
	"time"
)

var now = time.Now()

func split(s string, at int) (string, string) {
	if len(s) <= at {
		return s, ""
	}
	return s[:at], s[at:]
}

func formatDuration(d time.Duration) string {
	s := d.String()
	if strings.Contains(s, ".") {
		s = s[:strings.Index(s, ".")] + "s"
	}
	if strings.HasPrefix(s, "0h") {
		s = s[2:]
	}
	s = strings.ReplaceAll(s, "h0m", "h")
	s = strings.ReplaceAll(s, "m0s", "m")
	return s
}

type FormatFun func(event *gocal.Event, isCurrent bool) []string

func formatFancy(e *gocal.Event, isCurrent bool) []string {
	var content []string
	content = append(content, color.CyanString("|"))

	// print name
	if isCurrent {
		content = append(content, ">")
		content = append(content, color.New(color.FgBlack, color.BgHiMagenta).Sprintf(" %s ", e.Summary))
	} else {
		content = append(content, color.BlueString(e.Summary))

		content = append(content, fmt.Sprintf("[%s]",
			e.End.Sub(*e.Start)))
	}

	// print time
	content = append(content, "from")
	content = append(content, color.YellowString(e.Start.Format("15:04")))

	content = append(content, "to")
	content = append(content, color.YellowString(e.End.Format("15:04")))

	if isCurrent {
		// calculate total time in minutes
		totalMins := e.End.Sub(*e.Start).Minutes()
		// calculate elapsed time in minutes
		currentMins := now.Sub(*e.Start).Minutes()

		// progress of the lecture from 0 to 1
		progress := float32(currentMins) / float32(totalMins)

		progressBarPrompt := formatDuration(now.Sub(*e.Start)) + " / " + formatDuration(e.End.Sub(*e.Start))

		progressBarTotalLength := int(math.Max(20, float64(len(progressBarPrompt))))
		progressBarCurrentLength := int(progress * float32(progressBarTotalLength))
		progressBarPromptBefore, progressBarPromptAfter := split(progressBarPrompt, progressBarCurrentLength)

		progressBarColorActive := color.New(color.FgBlack, color.BgHiWhite)
		content = append(content,
			progressBarColorActive.Sprintf("[ %s%s",
				progressBarPromptBefore,
				strings.Repeat(" ", progressBarCurrentLength-len(progressBarPromptBefore)))+
				fmt.Sprintf("%s%s ]",
					progressBarPromptAfter,
					strings.Repeat(" ", progressBarTotalLength-progressBarCurrentLength-len(progressBarPromptAfter))))

		// show when lecture ends
		content = append(content, color.GreenString(fmt.Sprintf("(%s remaining)",
			formatDuration(e.End.Sub(now)))))
	} else if e.Start.After(now) {
		// show when lecture starts
		content = append(content, color.WhiteString(fmt.Sprintf("(in %s)",
			formatDuration(e.Start.Sub(now)))))
	} else {
		// show when the lecture finished
		content = append(content, color.WhiteString(fmt.Sprintf("(%s ago)",
			formatDuration(now.Sub(*e.End)))))
	}
	return content
}

func formatShell(e *gocal.Event, isCurrent bool) (content []string) {
	content = append(content, e.Summary)
	if isCurrent {
		content = append(content, fmt.Sprintf("[%s remaining]",
			formatDuration(e.End.Sub(now))))
	} else if now.After(*e.End) {
		content = append(content, fmt.Sprintf("[%s ago]",
			formatDuration(now.Sub(*e.End))))
	} else {
		content = append(content, fmt.Sprintf("[in %s]",
			formatDuration(e.Start.Sub(now))))
	}
	return
}

func main() {

	var (
		currentOnly bool
		formatter   FormatFun
	)

	formatters := map[string]FormatFun{
		"default": formatFancy,
		"shell":   formatShell,
	}

	if len(os.Args) > 1 {
		if f, ok := formatters[os.Args[1]]; ok {
			formatter = f
		} else {
			panic("cannot find formatter " + os.Args[1])
		}
	} else {
		formatter = formatters["default"]
	}

	if len(os.Args) > 2 {
		currentOnly = os.Args[2] == "current"
	}

	path, ok := os.LookupEnv("ICAL_PATH")
	if !ok {
		panic("ICAL_PATH not set")
	}
	// read ical file
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	nowStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	nowEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.UTC)

	cal := gocal.NewParser(f)
	cal.Start, cal.End = &nowStart, &nowEnd
	if err := cal.Parse(); err != nil {
		panic(err)
	}

	type event struct {
		event  *gocal.Event
		prompt []string
	}
	var events []*event

	for _, e := range cal.Events {
		if e.Start == nil || e.End == nil {
			fmt.Println("- ", color.BlueString(e.Summary))
			continue
		}
		isCurrent := now.After(*e.Start) && now.Before(*e.End)
		if currentOnly && !isCurrent {
			continue
		}
		events = append(events, &event{
			event:  &e,
			prompt: formatter(&e, isCurrent),
		})
	}

	// sort by starting time
	sort.Slice(events, func(i, j int) bool {
		return events[i].event.Start.Before(*events[j].event.End)
	})

	for _, e := range events {
		fmt.Println(strings.Join(e.prompt, " "))
	}
}
