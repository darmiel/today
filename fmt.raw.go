package main

import "strings"

func createRawFormatter() (interface{}, error) {
	return func(data []byte) ([]string, error) {
		return strings.Split(string(data), "\n"), nil
	}, nil
}
