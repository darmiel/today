package main

import (
	"fmt"
	"github.com/apognu/gocal"
	"github.com/fatih/color"
	"github.com/ralf-life/engine/pkg/environ"
	"html/template"
	"math"
	"strings"
	"time"
)

type templateContext struct {
	Event       *gocal.Event
	Relative    string
	RelativeRAW string
	Progress    string
	Start       *environ.CtxTime
	End         *environ.CtxTime
	IsCurrent   bool
}

func createProgressBar(event *gocal.Event) string {
	// calculate total time in minutes
	totalMins := event.End.Sub(*event.Start).Minutes()

	// calculate elapsed time in minutes
	timeToStart := now.Sub(*event.Start)
	if timeToStart < 0 {
		timeToStart = time.Duration(0)
	}
	currentMins := timeToStart.Minutes()

	if currentMins >= totalMins {
		return "[ FINISHED             ]"
	}

	// progress of the lecture from 0 to 1
	progress := float32(currentMins) / float32(totalMins)

	progressBarPrompt := formatDuration(timeToStart) + " / " + formatDuration(event.End.Sub(*event.Start))

	progressBarTotalLength := int(math.Max(20, float64(len(progressBarPrompt))))
	progressBarCurrentLength := int(progress * float32(progressBarTotalLength))
	progressBarPromptBefore, progressBarPromptAfter := split(progressBarPrompt, progressBarCurrentLength)

	progressBarColorActive := color.New(color.FgBlack, color.BgHiWhite)

	progressBarContentPrefix := fmt.Sprintf("[ %s%s",
		progressBarPromptBefore,
		strings.Repeat(" ", progressBarCurrentLength-len(progressBarPromptBefore)))
	if currentMins > 0 {
		progressBarContentPrefix = progressBarColorActive.Sprint(progressBarContentPrefix)
	}
	return progressBarContentPrefix +
		fmt.Sprintf("%s%s ]",
			progressBarPromptAfter,
			strings.Repeat(" ", progressBarTotalLength-progressBarCurrentLength-len(progressBarPromptAfter)))
}

func createTemplateContext(tpl string, ctx *FormatContext) *templateContext {
	var (
		relative    string
		relativeRaw string
	)
	if ctx.isCurrent {
		relativeRaw = formatDuration(ctx.event.End.Sub(now))
		relative = fmt.Sprintf("%s remaining", relativeRaw)
	} else if now.After(*ctx.event.End) {
		relativeRaw = formatDuration(now.Sub(*ctx.event.End))
		relative = fmt.Sprintf("%s ago", relative)
	} else {
		relativeRaw = formatDuration(ctx.event.Start.Sub(now))
		relative = fmt.Sprintf("in %s", relativeRaw)
	}
	var progress string
	if strings.Contains(tpl, "Progress") {
		progress = createProgressBar(ctx.event)
	}
	tStart, tEnd := environ.NewTime(*ctx.event.Start), environ.NewTime(*ctx.event.End)
	return &templateContext{
		Event:       ctx.event,
		Relative:    relative,
		RelativeRAW: relativeRaw,
		Progress:    progress,
		Start:       &tStart,
		End:         &tEnd,
		IsCurrent:   now.After(*ctx.event.Start) && now.Before(*ctx.event.End),
	}
}

func createTemplateFormatter(tpl string) func() (interface{}, error) {
	return func() (interface{}, error) {
		t, err := template.New("main").Parse(tpl)
		if err != nil {
			return nil, err
		}
		return func(ctx *FormatContext) ([]string, error) {
			var bob strings.Builder
			if err := t.Execute(&bob, createTemplateContext(tpl, ctx)); err != nil {
				return nil, err
			}
			return strings.Split(bob.String(), " "), nil
		}, nil
	}
}
