package apitest

import (
	"net/url"
	"testing"

	"github.com/raharper/ocidist/pkg/api"
	"github.com/stretchr/testify/assert"
)

func TestOCIDistRepoBasePath(t *testing.T) {
	assert := assert.New(t)
	cases := []struct {
		input    string
		expected string
	}{
		{"ocidist://localhost:5000/ubuntu/jammy:latest", "http://localhost:5000"},
		{"docker://localhost:5000/ubuntu/jammy:latest", "http://localhost:5000"},
		{"http://localhost:5000/ubuntu/jammy:latest", "http://localhost:5000"},
		{"docker://dockeruser:dockerpass@localhost:5000/ubuntu/jammy:latest", "http://dockeruser:dockerpass@localhost:5000"},
	}
	for _, c := range cases {
		url, err := url.Parse(c.input)
		assert.Nil(err)
		o, err := api.NewOCIDistRepo(url, &api.OCIAPIConfig{Debug: true})
		assert.Nil(err)
		assert.NotNil(o)
		assert.Equalf(c.expected, o.BasePath(), "input: '%s' - expected '%s' got '%s'", c.input, c.expected, o.BasePath())
	}
}

func TestOCIDistRepoBasePathErrors(t *testing.T) {
	assert := assert.New(t)
	cases := []struct {
		input    string
		expected string
	}{
		{"oci:ubuntu/jammy:latest", ""},
		{"ubuntu:latest", ""},
	}
	for _, c := range cases {
		url, err := url.Parse(c.input)
		assert.Nil(err)
		o, err := api.NewOCIDistRepo(url, &api.OCIAPIConfig{Debug: true})
		assert.Nil(err)
		assert.NotNil(o)
		assert.Equalf(c.expected, o.BasePath(), "input: '%s' - expected '%s' got '%s'", c.input, c.expected, o.BasePath())
	}
}

func TestOCIDistRepoPath(t *testing.T) {
	assert := assert.New(t)
	cases := []struct {
		input    string
		expected string
	}{
		{"ocidist://localhost:5000/ubuntu/jammy:latest", "ubuntu/jammy"},
		{"docker://localhost:5000/ubuntu/jammy:latest", "ubuntu/jammy"},
		{"http://localhost:5000/ubuntu/jammy:latest", "ubuntu/jammy"},
		{"docker://dockeruser:dockerpass@localhost:5000/ubuntu/jammy:latest", "ubuntu/jammy"},
	}
	for _, c := range cases {
		url, err := url.Parse(c.input)
		assert.Nil(err)
		o, err := api.NewOCIDistRepo(url, &api.OCIAPIConfig{Debug: true})
		assert.Nil(err)
		assert.NotNil(o)
		assert.Equalf(c.expected, o.RepoPath(), "input: '%s' - expected '%s' got '%s'", c.input, c.expected, o.RepoPath())
	}
}
