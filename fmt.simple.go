package main

func createSimpleFormatter() (interface{}, error) {
	return createTemplateFormatter("{{ .Event.Summary }} [{{ .Relative }}]")()
}
