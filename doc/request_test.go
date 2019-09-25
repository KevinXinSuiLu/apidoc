// SPDX-License-Identifier: MIT

package doc

import (
	"encoding/xml"
	"testing"

	"github.com/issue9/assert"
)

var _ xml.Unmarshaler = &Request{}

func TestRequest_UnmarshalXML(t *testing.T) {
	a := assert.New(t)

	obj := &Request{
		Type:     String,
		Mimetype: "json",
	}
	str := `<Request type="string" mimetype="json"></Request>`

	data, err := xml.Marshal(obj)
	a.NotError(err).Equal(string(data), str)

	obj1 := &Request{}
	a.NotError(xml.Unmarshal([]byte(str), obj1))
	a.Equal(obj1, obj)

	// 正常
	str = `<Request deprecated="1.1.1" type="object" array="true" mimetype="json">
		<param name="name" type="string" />
		<param name="sex" type="string">
			<enum value="male">Male</enum>
			<enum value="female">Female</enum>
		</param>
		<param name="age" type="number" />
	</Request>`
	a.NotError(xml.Unmarshal([]byte(str), obj1)).
		True(obj1.Array).
		Equal(obj1.Type, Object).
		Equal(obj1.Deprecated, "1.1.1").
		Equal(3, len(obj1.Items))

	// 少 name
	str = `<Request url="url" mimetype="json">desc</Request>`
	a.Error(xml.Unmarshal([]byte(str), obj1))

	// 少 type
	str = `<Request name="v1" mimetype="json"></Request>`
	a.Error(xml.Unmarshal([]byte(str), obj1))

	// 少 mimetype
	str = `<Request type="string"></Request>`
	a.Error(xml.Unmarshal([]byte(str), obj1))

	// type=object，且没有子项
	str = `<Request type="Object" mimetype="json"></Request>`
	a.Error(xml.Unmarshal([]byte(str), obj1))

	// 相同的子项
	str = `<Request type="Object" mimetype="json">
		<param name="n1" type="string" />
		<param name="n1" type="number" />
	</Request>`
	a.Error(xml.Unmarshal([]byte(str), obj1))

	// 语法错误
	str = `<Request deprecated="x.1.1" mimetype="json">text</Request>`
	a.Error(xml.Unmarshal([]byte(str), obj1))
}

func TestRequest_UnmarshalXML_enum(t *testing.T) {
	a := assert.New(t)

	obj := &Request{}
	str := `<Request name="sex" type="string" mimetype="json">
			<enum value="male">Male</enum>
			<enum value="female">Female</enum>
	</Request>`
	a.NotError(xml.Unmarshal([]byte(str), obj)).
		False(obj.Array).
		True(obj.IsEnum()).
		Equal(obj.Type, String).
		Equal(2, len(obj.Enums))

	// 枚举中存在相同值
	obj = &Request{}
	str = `<Request name="sex" type="string" mimetype="json">
			<enum value="female">Male</enum>
			<enum value="female">Female</enum>
	</Request>`
	a.Error(xml.Unmarshal([]byte(str), obj))
}