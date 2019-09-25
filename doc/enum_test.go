// SPDX-License-Identifier: MIT

package doc

import (
	"encoding/xml"
	"testing"

	"github.com/issue9/assert"
)

var (
	_ xml.Unmarshaler = &Enum{}
)

func TestEnum_UnmarshalXML(t *testing.T) {
	a := assert.New(t)

	obj := &Enum{
		Value:       "text",
		Description: "<a>desc</a>",
	}
	str := `<Enum value="text"><![CDATA[<a>desc</a>]]></Enum>`

	data, err := xml.Marshal(obj)
	a.NotError(err).Equal(string(data), str)

	obj1 := &Enum{}
	a.NotError(xml.Unmarshal([]byte(str), obj1))
	a.Equal(obj1, obj)

	// 正常
	str = `<Enum value="url" deprecated="1.1.1">text</Enum>`
	a.NotError(xml.Unmarshal([]byte(str), obj1))

	// 少 value
	str = `<Enum url="url">desc</Enum>`
	a.Error(xml.Unmarshal([]byte(str), obj1))

	// 少 description
	str = `<Enum value="v1"></Enum>`
	a.Error(xml.Unmarshal([]byte(str), obj1))

	// 语法错误
	str = `<Enum value="url" deprecated="x.1.1">text</Enum>`
	a.Error(xml.Unmarshal([]byte(str), obj1))
}