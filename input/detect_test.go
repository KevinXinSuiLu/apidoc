// SPDX-License-Identifier: MIT

package input

import (
	"testing"

	"github.com/issue9/assert"
)

func TestDetect(t *testing.T) {
	a := assert.New(t)

	o, err := Detect("./testdata", true)
	a.NotError(err).NotEmpty(o)
	a.Equal(len(o), 2). // c and php
				Equal(o[0].Lang, "c++").
				Equal(o[1].Lang, "php")
}

func TestDetectLanguage(t *testing.T) {
	a := assert.New(t)
	exts := map[string]int{
		".h":     2,
		".c":     3,
		".swift": 1,
		".php":   2,
	}

	langs := detectLanguage(exts)
	a.Equal(len(langs), 3) // c++,php,swift
	a.Equal(langs[0].Name, "c++").
		Equal(langs[0].count, 5)
	a.Equal(langs[1].Name, "php").
		Equal(langs[1].count, 2)
	a.Equal(langs[2].Name, "swift").
		Equal(langs[2].count, 1)
}

func TestDetectExts(t *testing.T) {
	a := assert.New(t)

	files, err := detectExts("./testdata", false)
	a.NotError(err)
	a.Equal(len(files), 4)
	a.Equal(files[".php"], 1).Equal(files[".c"], 1)

	files, err = detectExts("./testdata", true)
	a.NotError(err)
	a.Equal(len(files), 5)
	a.Equal(files[".php"], 1).Equal(files[".1"], 3)
}
