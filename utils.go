package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/goyesql"
	"github.com/labstack/echo"
	"github.com/lib/pq"
)

var (

	// This replaces all special characters
	tagRegexp       = regexp.MustCompile(`[^a-z0-9\-\s]`)
	tagRegexpSpaces = regexp.MustCompile(`[\s]+`)
)

// ScanToStruct prepares a given set of Queries and assigns the resulting
// *sql.Stmt statements to the fields of a given struct, matching based on the name
// in the `query` tag in the struct field names.
func scanQueriesToStruct(obj interface{}, q goyesql.Queries, db *sqlx.DB) error {
	ob := reflect.ValueOf(obj)
	if ob.Kind() == reflect.Ptr {
		ob = ob.Elem()
	}

	if ob.Kind() != reflect.Struct {
		return fmt.Errorf("Failed to apply SQL statements to struct. Non struct type: %T", ob)
	}

	// Go through every field in the struct and look for it in the Args map.
	for i := 0; i < ob.NumField(); i++ {
		f := ob.Field(i)

		if f.IsValid() {
			if tag := ob.Type().Field(i).Tag.Get("query"); tag != "" && tag != "-" {
				// Extract the value of the `query` tag.
				var (
					tg   = strings.Split(tag, ",")
					name string
				)
				if len(tg) == 2 {
					if tg[0] != "-" && tg[0] != "" {
						name = tg[0]
					}
				} else {
					name = tg[0]
				}

				// Query name found in the field tag is not in the map.
				if _, ok := q[name]; !ok {
					return fmt.Errorf("query '%s' not found in query map", name)
				}

				if !f.CanSet() {
					return fmt.Errorf("query field '%s' is unexported", ob.Type().Field(i).Name)
				}

				switch f.Type().String() {
				case "string":
					// Unprepared SQL query.
					f.Set(reflect.ValueOf(q[name].Query))
				case "*sqlx.Stmt":
					// Prepared query.
					stmt, err := db.Preparex(q[name].Query)
					if err != nil {
						return fmt.Errorf("Error preparing query '%s': %v", name, err)
					}

					f.Set(reflect.ValueOf(stmt))
				}
			}
		}
	}

	return nil
}

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

// createThumbnail reads the file object and returns a smaller image
func createThumbnail(file *multipart.FileHeader) (*bytes.Reader, error) {
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()
	img, err := imaging.Decode(src)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error decoding image: %v", err))
	}
	t := imaging.Resize(img, thumbnailSize, 0, imaging.Lanczos)
	// Encode the image into a byte slice as PNG.
	var buf bytes.Buffer
	err = imaging.Encode(&buf, t, imaging.PNG)
	if err != nil {
		log.Fatal(err)
	}
	return bytes.NewReader(buf.Bytes()), nil
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
