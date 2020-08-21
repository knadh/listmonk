package dav

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPutFile(t *testing.T) {
	c, err := NewDavStore(Opts{
		Endpoint: "https://webdav.yandex.com",
		Password: "",
		Username: "",
		RootPath: "/",
		Headers: map[string]string{
			"Authorization": "OAuth TestToken-TestToken",
		},
	})
	require.NoError(t, err)

	body, err := ioutil.ReadFile("dav.go")
	require.NoError(t, err)

	_, err = c.Put("dav.go", "", bytes.NewReader(body))
	require.NoError(t, err)
}

func TestGetFile(t *testing.T) {
	c, err := NewDavStore(Opts{
		Endpoint: "https://webdav.yandex.com",
		Password: "",
		Username: "",
		RootPath: "/",
		Headers: map[string]string{
			"Authorization": "OAuth TestToken",
		},
	})
	require.NoError(t, err)

	body := c.Get("dav.go")
	require.NotEmpty(t, body)
}

func TestDeleteFile(t *testing.T) {
	c, err := NewDavStore(Opts{
		Endpoint: "https://webdav.yandex.com",
		Password: "",
		Username: "",
		RootPath: "/",
		Headers: map[string]string{
			"Authorization": "OAuth TestToken-TestToken",
		},
	})
	require.NoError(t, err)

	err = c.Delete("/dav.go")
	require.NoError(t, err)
}
