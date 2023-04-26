package main

import (
	"fmt"
	"github.com/fatih/color"
)

func createDefaultFormatter() (FormatFun, error) {
	return func(ctx *FormatContext) ([]string, error) {
		e := ctx.event
		var content []string

		// print name
		if ctx.isCurrent {
			content = append(content, ">")
			content = append(content, color.New(color.FgBlack, color.BgHiMagenta).Sprintf(" %s ", e.Summary))
		} else {
			content = append(content, " ")
			content = append(content, color.BlueString(e.Summary))
			content = append(content, fmt.Sprintf("[%s]",
				e.End.Sub(*e.Start)))
		}

		// print time
		content = append(content, "from")
		content = append(content, color.YellowString(e.Start.Format("15:04")))

		content = append(content, "to")
		content = append(content, color.YellowString(e.End.Format("15:04")))

		if ctx.isCurrent {
			content = append(content, createProgressBar(e))
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
		return content, nil
	}, nil
}
