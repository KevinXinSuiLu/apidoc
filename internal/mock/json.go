// SPDX-License-Identifier: MIT

package mock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/caixw/apidoc/v6/core"
	"github.com/caixw/apidoc/v6/internal/locale"
	"github.com/caixw/apidoc/v6/spec"
)

// 缩进的字符串
const indent = "    "

type jsonValidator struct {
	param   *spec.Param
	decoder *json.Decoder

	// 按顺序表示的状态
	// 可以是 [ 表示在数组中，{ 表示在对象，: 表示下一个值必须是属性，空格表示其它状态
	states []byte

	// 按顺序保存变量名称
	names []string
}

type jsonBuilder struct {
	buf          *bytes.Buffer
	err          error
	deep         int
	indentString string
}

func validJSON(p *spec.Request, content []byte) error {
	if p == nil {
		if bytes.Equal(content, []byte("null")) {
			return nil
		}
		return core.NewLocaleError("", "", 0, locale.ErrInvalidFormat)
	}

	if (p.Type == spec.None) && len(content) == 0 {
		return nil
	}

	if !json.Valid(content) {
		return core.NewLocaleError("", "", 0, locale.ErrInvalidFormat)
	}

	validator := &jsonValidator{
		param:   p.Param(),
		decoder: json.NewDecoder(bytes.NewReader(content)),
		states:  []byte{}, // 状态有默认值
		names:   []string{},
	}

	return validator.valid()
}

func (validator *jsonValidator) valid() error {
	for {
		token, err := validator.decoder.Token()
		if err == io.EOF && token == nil { // 正常结束
			return nil
		}

		if err != nil {
			return err
		}

		if token == nil { // 对应 JSON null
			if err = validator.validValue("", nil); err != nil {
				return err
			}
			validator.popState()
			validator.popName()
		}

		switch v := token.(type) {
		case string: // json string
			switch validator.state() {
			case ':': // 字符串类型的值
				err = validator.validValue(spec.String, v)
				validator.popState()
				validator.popName()
			case '[':
				err = validator.validValue(spec.String, v)
			default: // 属性名
				validator.pushState(':')
				validator.pushName(v)
			}

			if err != nil {
				return err
			}
		case json.Delim: // [、]、{、}
			switch v {
			case '[':
				validator.pushState('[')
			case ']':
				validator.popName()

				validator.popState()
				if validator.state() == ':' { // {xx: [] } 类似这种格式，需要同时弹出两个状态
					validator.popState()
				}
			case '{':
				validator.pushState('{')
			case '}':
				validator.popName()

				validator.popState()
				if validator.state() == ':' {
					validator.popState()
				}
			}
		case bool: // json bool
			err = validator.validValue(spec.Bool, v)
			if validator.state() != '[' {
				validator.popState()
				validator.popName()
			}
		case float64, json.Number: // json number
			err = validator.validValue(spec.Number, v)
			if validator.state() != '[' { // 只有键值对结束时，才弹出键名
				validator.popState()
				validator.popName()
			}
		}

		if err != nil {
			return err
		}
	}
}

// 如果 t == "" 表示不需要验证类型，比如 null 可以赋值给任何类型
func (validator *jsonValidator) validValue(t spec.Type, v interface{}) error {
	field := strings.Join(validator.names, ".")

	p := validator.find()
	if p == nil {
		return core.NewLocaleError("", field, 0, locale.ErrNotFound)
	}

	if t == "" {
		return nil
	}

	if p.Type != t {
		return core.NewLocaleError("", field, 0, locale.ErrInvalidFormat)
	}

	if p.IsEnum() {
		for _, enum := range p.Enums {
			if enum.Value == fmt.Sprint(v) {
				return nil
			}
		}
		return core.NewLocaleError("", field, 0, locale.ErrInvalidValue)
	}

	return nil
}

// 返回当前的状态
func (validator *jsonValidator) state() byte {
	if len(validator.states) > 0 {
		return validator.states[len(validator.states)-1]
	}
	return 0
}

func (validator *jsonValidator) pushState(state byte) *jsonValidator {
	validator.states = append(validator.states, state)
	return validator
}

func (validator *jsonValidator) popState() *jsonValidator {
	if len(validator.states) > 0 {
		validator.states = validator.states[:len(validator.states)-1]
	}
	return validator
}

func (validator *jsonValidator) pushName(name string) *jsonValidator {
	validator.names = append(validator.names, name)
	return validator
}

func (validator *jsonValidator) popName() *jsonValidator {
	if len(validator.names) > 0 {
		validator.names = validator.names[:len(validator.names)-1]
	}
	return validator
}

// 如果 names 为空，返回 validator.param
func (validator *jsonValidator) find() *spec.Param {
	p := validator.param
	for _, name := range validator.names {
		found := false
		for _, pp := range p.Items {
			if pp.Name == name {
				p = pp
				found = true
				break
			}
		}

		if !found {
			return nil
		}
	}

	return p
}

func (builder *jsonBuilder) writeIndent() *jsonBuilder {
	if builder.err != nil {
		return builder
	}
	_, builder.err = builder.buf.WriteString(builder.indentString)
	return builder
}

func (builder *jsonBuilder) incrIndent() *jsonBuilder {
	builder.deep++
	builder.indentString = strings.Repeat(indent, builder.deep)
	return builder
}

func (builder *jsonBuilder) decrIndent() *jsonBuilder {
	builder.deep--
	builder.indentString = strings.Repeat(indent, builder.deep)
	return builder
}

func (builder *jsonBuilder) writeStrings(str ...string) *jsonBuilder {
	if builder.err != nil {
		return builder
	}

	for _, s := range str {
		_, builder.err = builder.buf.WriteString(s)
		if builder.err != nil {
			break
		}
	}

	return builder
}

// v 只能是基本类型
func (builder *jsonBuilder) writeValue(v interface{}) *jsonBuilder {
	if builder.err != nil {
		return builder
	}

	vv, err := json.Marshal(v)
	if err != nil {
		builder.err = err
		return builder
	}

	_, builder.err = builder.buf.Write(vv)
	return builder
}

func buildJSON(p *spec.Request) ([]byte, error) {
	if p != nil && p.Type == spec.None {
		return nil, nil
	}

	builder := &jsonBuilder{
		buf: new(bytes.Buffer),
	}

	if err := writeJSON(builder, p.Param(), true); err != nil {
		return nil, err
	}

	return builder.buf.Bytes(), nil
}

func writeJSON(builder *jsonBuilder, p *spec.Param, chkArray bool) error {
	if p == nil {
		builder.writeValue(nil)
		return builder.err
	}

	if p.Array && chkArray {
		builder.writeStrings("[\n").incrIndent()

		size := generateSliceSize()
		last := size - 1
		for i := 0; i < size; i++ {
			builder.writeIndent()

			if err := writeJSON(builder, p, false); err != nil {
				return err
			}

			if i < last {
				builder.writeStrings(",\n")
			} else {
				builder.writeStrings("\n")
			}
		}

		builder.decrIndent().writeIndent().writeStrings("]")
		return builder.err
	}

	switch p.Type {
	case spec.None:
		builder.writeValue(nil)
	case spec.Bool:
		builder.writeValue(generateBool())
	case spec.Number:
		builder.writeValue(generateNumber(p))
	case spec.String:
		builder.writeValue(generateString(p))
	case spec.Object:
		builder.writeStrings("{\n").incrIndent()

		last := len(p.Items) - 1
		for index, item := range p.Items {
			builder.writeIndent().writeStrings(`"`, item.Name, `"`, ": ")

			if err := writeJSON(builder, item, true); err != nil {
				return err
			}

			if index < last {
				builder.writeStrings(",\n")
			} else {
				builder.writeStrings("\n")
			}
		}

		builder.decrIndent().writeIndent().writeStrings("}")
	}

	return builder.err
}
