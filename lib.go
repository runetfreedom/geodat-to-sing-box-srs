package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func readFile(path string) ([]byte, error) {
	switch {
	case strings.HasPrefix(strings.ToLower(path), "http://"), strings.HasPrefix(strings.ToLower(path), "https://"):
		resp, err := http.Get(path)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("failed to get remote file %s, http status code %d", path, resp.StatusCode)
		}

		return io.ReadAll(resp.Body)

	default:
		return os.ReadFile(path)
	}
}
