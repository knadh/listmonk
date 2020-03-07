package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/lib/pq"
)

var (

	// This replaces all special characters
	tagRegexp       = regexp.MustCompile(`[^a-z0-9\-\s]`)
	tagRegexpSpaces = regexp.MustCompile(`[\s]+`)
)

// validateMIME is a helper function to validate uploaded file's MIME type
// against the slice of MIME types is given.
func validateMIME(typ string, mimes []string) (ok bool) {
	if len(mimes) > 0 {
		var (
			ok = false
		)
		for _, m := range mimes {
			if typ == m {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

// generateFileName appends the incoming file's name with a small random hash.
func generateFileName(fName string) string {
	name := strings.TrimSpace(fName)
	if name == "" {
		name, _ = generateRandomString(10)
	}
	return name
}

// Given an error, pqErrMsg will try to return pq error details
// if it's a pq error.
func pqErrMsg(err error) string {
	if err, ok := err.(*pq.Error); ok {
		if err.Detail != "" {
			return fmt.Sprintf("%s. %s", err, err.Detail)
		}
	}
	return err.Error()
}

// normalizeTags takes a list of string tags and normalizes them by
// lowercasing and removing all special characters except for dashes.
func normalizeTags(tags []string) []string {
	var (
		out   []string
		space = []byte(" ")
		dash  = []byte("-")
	)

	for _, t := range tags {
		rep := bytes.TrimSpace(tagRegexp.ReplaceAll(bytes.ToLower([]byte(t)), space))
		rep = tagRegexpSpaces.ReplaceAll(rep, dash)

		if len(rep) > 0 {
			out = append(out, string(rep))
		}
	}
	return out
}

// makeMsgTpl takes a page title, heading, and message and returns
// a msgTpl that can be rendered as a HTML view. This is used for
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
func parseStringIDs(s []string) ([]int64, error) {
	vals := make([]int64, 0, len(s))
	for _, v := range s {
		i, err := strconv.ParseInt(v, 10, 64)
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
