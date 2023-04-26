package main

func createSimpleFormatter() (FormatFun, error) {
	return createTemplateFormatter("{{ .Event.Summary }} [{{ .Relative }}]")()
}
