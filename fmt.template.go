package main

import (
	"fmt"
	"github.com/apognu/gocal"
	"github.com/fatih/color"
	"html/template"
	"math"
	"strings"
)

type templateContext struct {
	Event    *gocal.Event
	Relative string
	Progress string
}

func createProgressBar(event *gocal.Event) string {
	// calculate total time in minutes
	totalMins := event.End.Sub(*event.Start).Minutes()
	// calculate elapsed time in minutes
	currentMins := now.Sub(*event.Start).Minutes()

	if currentMins >= totalMins {
		return "[ FINISHED             ]"
	}

	// progress of the lecture from 0 to 1
	progress := float32(currentMins) / float32(totalMins)

	progressBarPrompt := formatDuration(now.Sub(*event.Start)) + " / " + formatDuration(event.End.Sub(*event.Start))

	progressBarTotalLength := int(math.Max(20, float64(len(progressBarPrompt))))
	progressBarCurrentLength := int(progress * float32(progressBarTotalLength))
	progressBarPromptBefore, progressBarPromptAfter := split(progressBarPrompt, progressBarCurrentLength)

	progressBarColorActive := color.New(color.FgBlack, color.BgHiWhite)
	return progressBarColorActive.Sprintf("[ %s%s",
		progressBarPromptBefore,
		strings.Repeat(" ", progressBarCurrentLength-len(progressBarPromptBefore))) +
		fmt.Sprintf("%s%s ]",
			progressBarPromptAfter,
			strings.Repeat(" ", progressBarTotalLength-progressBarCurrentLength-len(progressBarPromptAfter)))
}

func createTemplateContext(tpl string, ctx *FormatContext) *templateContext {
	var relative string
	if ctx.isCurrent {
		relative = fmt.Sprintf("%s remaining", formatDuration(ctx.event.End.Sub(now)))
	} else if now.After(*ctx.event.End) {
		relative = fmt.Sprintf("%s ago", formatDuration(now.Sub(*ctx.event.End)))
	} else {
		relative = fmt.Sprintf("in %s", formatDuration(ctx.event.Start.Sub(now)))
	}
	var progress string
	if strings.Contains(tpl, "Progress") {
		progress = createProgressBar(ctx.event)
	}
	return &templateContext{
		Event:    ctx.event,
		Relative: relative,
		Progress: progress,
	}
}

func createTemplateFormatter(tpl string) FormatInitFun {
	return func() (FormatFun, error) {
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
