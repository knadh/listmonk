package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/goyesql"
	"github.com/labstack/echo"
	"github.com/lib/pq"
)

const tmpFilePrefix = "listmonk"

var (
	// This matches filenames, sans extensions, of the format
	// filename_(number). The number is incremented in case
	// new file uploads conflict with existing filenames
	// on the filesystem.
	fnameRegexp, _ = regexp.Compile(`(.+?)_([0-9]+)$`)

	// This replaces all special characters
	tagRegexp, _       = regexp.Compile(`[^a-z0-9\-\s]`)
	tagRegexpSpaces, _ = regexp.Compile(`[\s]+`)
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

// uploadFile is a helper function on top of echo.Context for processing file uploads.
// It allows copying a single file given the incoming file field name.
// If the upload directory dir is empty, the file is copied to the system's temp directory.
// If name is empty, the incoming file's name along with a small random hash is used.
// When a slice of MIME types is given, the uploaded file's MIME type is validated against the list.
func uploadFile(key string, dir, name string, mimes []string, c echo.Context) (string, error) {
	file, err := c.FormFile(key)
	if err != nil {
		return "", echo.NewHTTPError(http.StatusBadRequest,
			fmt.Sprintf("Invalid file uploaded: %v", err))
	}

	// Check MIME type.
	if len(mimes) > 0 {
		var (
			typ = file.Header.Get("Content-type")
			ok  = false
		)
		for _, m := range mimes {
			if typ == m {
				ok = true
				break
			}
		}

		if !ok {
			return "", echo.NewHTTPError(http.StatusBadRequest,
				fmt.Sprintf("Unsupported file type (%s) uploaded.", typ))
		}
	}

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// There's no upload directory. Use a tempfile.
	var out *os.File
	if dir == "" {
		o, err := ioutil.TempFile("", tmpFilePrefix)
		if err != nil {
			return "", echo.NewHTTPError(http.StatusInternalServerError,
				fmt.Sprintf("Error copying uploaded file: %v", err))
		}
		out = o
		name = o.Name()
	} else {
		// There's no explicit name. Use the one posted in the HTTP request.
		if name == "" {
			name = strings.TrimSpace(file.Filename)
			if name == "" {
				name, _ = generateRandomString(10)
			}
		}
		name = assertUniqueFilename(dir, name)

		o, err := os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
		if err != nil {
			return "", echo.NewHTTPError(http.StatusInternalServerError,
				fmt.Sprintf("Error copying uploaded file: %v", err))
		}

		out = o
	}
	defer out.Close()

	if _, err = io.Copy(out, src); err != nil {
		return "", echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error copying uploaded file: %v", err))
	}

	return name, nil
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

// assertUniqueFilename takes a file path and check if it exists on the disk. If it doesn't,
// it returns the same name and if it does, it adds a small random hash to the filename
// and returns that.
func assertUniqueFilename(dir, fileName string) string {
	var (
		ext  = filepath.Ext(fileName)
		base = fileName[0 : len(fileName)-len(ext)]
		num  = 0
	)

	for {
		// There's no name conflict.
		if _, err := os.Stat(filepath.Join(dir, fileName)); os.IsNotExist(err) {
			return fileName
		}

		// Does the name match the _(num) syntax?
		r := fnameRegexp.FindAllStringSubmatch(fileName, -1)
		if len(r) == 1 && len(r[0]) == 3 {
			num, _ = strconv.Atoi(r[0][2])
		}
		num++

		fileName = fmt.Sprintf("%s_%d%s", base, num, ext)
	}
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

// makeErrorTpl takes error details and returns an errorTpl
// with the error details applied to be rendered in an HTML view.
func makeErrorTpl(pageTitle, heading, desc string) errorTpl {
	if heading == "" {
		heading = pageTitle
	}
	err := errorTpl{}
	err.Title = pageTitle
	err.ErrorTitle = heading
	err.ErrorMessage = desc

	return err
}

// parseStringIDs takes a slice of numeric string IDs and
// parses each number into an int64 and returns a slice of the
// resultant values.
func parseStringIDs(s []string) ([]int64, error) {
	var vals []int64

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
