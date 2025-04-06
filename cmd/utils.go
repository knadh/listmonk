package main

import (
	"crypto/rand"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var (
	regexpSpaces = regexp.MustCompile(`[\s]+`)
)

// inArray checks if a string is present in a list of strings.
func inArray(val string, vals []string) (ok bool) {
	return slices.Contains(vals, val)
}

// makeFilename sanitizes a filename (user supplied upload filenames).
func makeFilename(fName string) string {
	name := strings.TrimSpace(fName)
	if name == "" {
		name, _ = generateRandomString(10)
	}
	// replace whitespace with "-"
	name = regexpSpaces.ReplaceAllString(name, "-")
	return filepath.Base(name)
}

// appendSuffixToFilename adds a string suffix to the filename while keeping the file extension.
func appendSuffixToFilename(filename, suffix string) string {
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	return fmt.Sprintf("%s_%s%s", name, suffix, ext)
}

// makeMsgTpl takes a page title, heading, and message and returns
// a msgTpl that can be rendered as an HTML view. This is used for
// rendering arbitrary HTML views with error and success messages.
func makeMsgTpl(pageTitle, heading, msg string) msgTpl {
	if heading == "" {
		heading = pageTitle
	}
	err := msgTpl{}
	err.Title = pageTitle
	err.MessageTitle = heading
	err.Message = msg
	return err
}

// parseStringIDs takes a slice of numeric string IDs and
// parses each number into an int64 and returns a slice of the
// resultant values.
func parseStringIDs(s []string) ([]int, error) {
	vals := make([]int, 0, len(s))
	for _, v := range s {
		i, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}

		if i < 1 {
			return nil, fmt.Errorf("%d is not a valid ID", i)
		}

		vals = append(vals, i)
	}

	return vals, nil
}

// generateRandomString generates a cryptographically random, alphanumeric string of length n.
func generateRandomString(n int) (string, error) {
	const dictionary = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	for k, v := range bytes {
		bytes[k] = dictionary[v%byte(len(dictionary))]
	}

	return string(bytes), nil
}

// strHasLen checks if the given string has a length within min-max.
func strHasLen(str string, min, max int) bool {
	return len(str) >= min && len(str) <= max
}

// getQueryInts parses the list of given query param values into ints.
func getQueryInts(param string, qp url.Values) ([]int, error) {
	var out []int
	if vals, ok := qp[param]; ok {
		for _, v := range vals {
			if v == "" {
				continue
			}

			listID, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			out = append(out, listID)
		}
	}

	return out, nil
}
