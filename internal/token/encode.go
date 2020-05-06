// SPDX-License-Identifier: MIT

package token

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"reflect"

	"github.com/issue9/is"
)

// Encoder 将元素内容编码成 XML 内容
type Encoder interface {
	// 仅需要返回元素内容的 XML 编码，不需要包含本身的标签和属性。
	EncodeXML() (string, error)
}

// AttrEncoder 将属性值编码成符合 XML 规范的值
type AttrEncoder interface {
	// 仅需要返回属性的 XML 表示，不需要包含属性值的引号字符。
	EncodeXMLAttr() (string, error)
}

var (
	attrEncoderType = reflect.TypeOf((*AttrEncoder)(nil)).Elem()
	encoderType     = reflect.TypeOf((*Encoder)(nil)).Elem()
)

// Encode 将 v 转换成 XML 内容
func Encode(indent string, v interface{}) ([]byte, error) {
	rv := parseRootElement(v)
	buf := new(bytes.Buffer)
	e := xml.NewEncoder(buf)
	e.Indent("", indent)
	n := newNode(rv.name, rv.Value)

	if err := n.encode(e); err != nil {
		return nil, err
	}

	if err := e.Flush(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (n *node) encode(e *xml.Encoder) error {
	start, err := n.buildStartElement()
	if err != nil {
		return err
	}

	if n.cdata.IsValid() && !n.cdata.isOmitempty() {
		chardata, err := getElementValue(n.cdata.Value)
		if err != nil {
			return err
		}

		return e.EncodeElement(struct {
			string `xml:",cdata"`
		}{chardata}, start)
	}

	if n.content.IsValid() && !n.content.isOmitempty() {
		chardata, err := getElementValue(n.content.Value)
		if err != nil {
			return err
		}
		return e.EncodeElement(xml.CharData(chardata), start)
	}

	return n.encodeElements(e, start)
}

func (n *node) encodeElements(e *xml.Encoder, start xml.StartElement) (err error) {
	if err = e.EncodeToken(start); err != nil {
		return err
	}
	for _, v := range n.elems {
		if err := encodeElement(e, v); err != nil {
			return err
		}
	}

	return e.EncodeToken(xml.EndElement{Name: xml.Name{Local: n.name}})
}

func encodeElement(e *xml.Encoder, v value) (err error) {
	if v.isOmitempty() {
		return nil
	}

	var chardata string
	var found bool

	if v.CanInterface() && v.Type().Implements(encoderType) {
		if chardata, err = v.Interface().(Encoder).EncodeXML(); err != nil {
			return err
		}
		found = true
	} else if !found && v.CanAddr() {
		pv := v.Addr()
		if pv.CanInterface() && pv.Type().Implements(encoderType) {
			if chardata, err = pv.Interface().(Encoder).EncodeXML(); err != nil {
				return err
			}
			found = true
		}
	}
	if !found && isPrimitive(v.Value) {
		chardata = fmt.Sprint(v.Interface())
		found = true
	}

	if found {
		start := xml.StartElement{Name: xml.Name{Local: v.name}}
		if err := e.EncodeElement(xml.CharData(chardata), start); err != nil {
			return err
		}
		return nil
	}

	if v.Kind() == reflect.Array || v.Kind() == reflect.Slice {
		for i := 0; i < v.Len(); i++ {
			if err := encodeElement(e, initValue(v.name, v.Index(i), v.omitempty, v.usage)); err != nil {
				return err
			}
		}
		return nil
	}

	return newNode(v.name, v.Value).encode(e)
}

func (n *node) buildStartElement() (xml.StartElement, error) {
	start := xml.StartElement{
		Name: xml.Name{Local: n.name},
		Attr: make([]xml.Attr, 0, len(n.attrs)),
	}

	for _, v := range n.attrs {
		if v.isOmitempty() {
			continue
		}

		val, err := getAttributeValue(v.Value)
		if err != nil {
			return xml.StartElement{}, err
		}

		start.Attr = append(start.Attr, xml.Attr{
			Name:  xml.Name{Local: v.name},
			Value: val,
		})
	}

	return start, nil
}

func getAttributeValue(elem reflect.Value) (string, error) {
	if elem.CanInterface() && elem.Type().Implements(attrEncoderType) {
		return elem.Interface().(AttrEncoder).EncodeXMLAttr()
	} else if elem.CanAddr() {
		pv := elem.Addr()
		if pv.CanInterface() && pv.Type().Implements(attrEncoderType) {
			return pv.Interface().(AttrEncoder).EncodeXMLAttr()
		}
	}

	return fmt.Sprint(elem.Interface()), nil
}

// 获取 CData 和 String 的编码内容，适用于 content 和 cdata 节点类型。
func getElementValue(elem reflect.Value) (string, error) {
	elem = getRealValue(elem)
	if elem.CanInterface() {
		switch {
		case elem.Type() == cdataType:
			return elem.Interface().(CData).Value.Value, nil
		case elem.Type() == contentType:
			return elem.Interface().(String).Value, nil
		}
	} else if elem.CanAddr() {
		if pv := elem.Addr(); pv.CanInterface() {
			switch {
			case pv.Type() == cdataType:
				return elem.Interface().(CData).Value.Value, nil
			case pv.Type() == contentType:
				return pv.Interface().(String).Value, nil
			}
		}
	}

	panic(fmt.Sprintf("%s 只能是 CData 或是 String 类型", elem.Type()))
}

func (v value) isOmitempty() bool {
	return v.omitempty && is.Empty(v.Value.Interface(), true)
}
