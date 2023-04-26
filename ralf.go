package main

import (
	"errors"
	"fmt"
	ics "github.com/darmiel/golang-ical"
	"github.com/ralf-life/engine/actions"
	"github.com/ralf-life/engine/engine"
	"github.com/ralf-life/engine/model"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrSourceMissing = errors.New("cannot find source")
	ErrInvalidSource = errors.New("source protocol not supported")
	ErrCacheNotDir   = errors.New("cache must be a directory")
)

func isExpired(stat os.FileInfo, duration time.Duration) bool {
	if stat == nil {
		return true
	}
	return time.Now().After(stat.ModTime().Add(duration))
}

func loadFileSourceFromRALFProfile(sourceFilePath string) (io.ReadCloser, error) {
	return os.Open(sourceFilePath)
}

func loadHTTPSourceFromRALFProfile(definitionPath string, profile *model.Profile, verbose bool) (io.ReadCloser, error) {
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
			return nil, fmt.Errorf("cannot create cache directory '%s': %v", tempDir, err)
		}
	} else if stat != nil && !stat.IsDir() {
		return nil, ErrCacheNotDir
	}

	fileName := filepath.Join(tempDir, filepath.Base(definitionPath)+".cached.ics")
	var duration time.Duration
	if int64(profile.CacheDuration) > 0 {
		duration = time.Duration(profile.CacheDuration)
	} else {
		duration = 5 * time.Minute
	}

	// check if cache file already exists or expired
	if stat, pathErr := os.Stat(fileName); os.IsNotExist(pathErr) || isExpired(stat, duration) {
		if verbose {
			fmt.Printf("cache: needing to re-download file %s\n", fileName)
		}

		// get source contents from URL
		resp, err := http.Get(profile.Source)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return nil, fmt.Errorf("expected status code 200-299 but got %d", resp.StatusCode)
		}

		f, err := os.Create(fileName)
		if err != nil {
			return nil, fmt.Errorf("cannot create cache output file '%s': %v", fileName, err)
		}
		defer f.Close()

		if _, err = io.Copy(f, resp.Body); err != nil {
			return nil, fmt.Errorf("cannot copy response body contents to '%s': %v", fileName, err)
		}
	} else if verbose && stat != nil {
		fmt.Println("cache: using cached file, expires in:",
			formatDuration(stat.ModTime().Add(duration).Sub(time.Now())))
	}

	return os.Open(fileName)
}

func loadSourceFromRALFProfile(definitionPath string, profile *model.Profile, verbose bool) (io.ReadCloser, error) {
	u, err := url.Parse(profile.Source)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "http", "https":
		// load iCal file via http
		return loadHTTPSourceFromRALFProfile(definitionPath, profile, verbose)
	case "file":
		// load iCal file from system
		return loadFileSourceFromRALFProfile(u.Path)
	}
	return nil, ErrInvalidSource
}

func getRALFReader(iCalPath, ralfDefinitionPath string, enableDebug, ralfVerbose, verbose bool) (io.Reader, error) {
	rf, err := os.Open(ralfDefinitionPath)
	if err != nil {
		return nil, fmt.Errorf("cannot open RALF-definition '%s': '%v'", ralfDefinitionPath, err)
	}
	defer rf.Close()

	// parse RALF profile from ralfDefinitionPath
	var profile model.Profile
	dec := yaml.NewDecoder(rf)
	dec.KnownFields(true)
	if err = dec.Decode(&profile); err != nil {
		return nil, err
	}

	cp := engine.ContextFlow{
		Profile:     &profile,
		Context:     make(map[string]interface{}),
		EnableDebug: enableDebug,
		Verbose:     ralfVerbose,
	}

	// load calendar contents from RALF-source
	var r io.ReadCloser
	if iCalPath != "" {
		if r, err = loadFileSourceFromRALFProfile(iCalPath); err != nil {
			return nil, err
		}
	} else if profile.Source != "" {
		if r, err = loadSourceFromRALFProfile(ralfDefinitionPath, &profile, verbose); err != nil {
			return nil, err
		}
	} else {
		return nil, ErrSourceMissing
	}
	defer r.Close()

	cal, err := ics.ParseCalendar(r)
	if err != nil {
		return nil, fmt.Errorf("cannot parse calendar: %v", err)
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
