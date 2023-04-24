package main

import (
	"fmt"
)

func formatSimple(ctx *FormatContext) (content []string) {
	e := ctx.event
	content = append(content, e.Summary)
	if ctx.isCurrent {
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
