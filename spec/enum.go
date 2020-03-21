// SPDX-License-Identifier: MIT

package spec

import (
	"encoding/xml"

	"github.com/caixw/apidoc/v6/core"
	"github.com/caixw/apidoc/v6/internal/locale"
)

// Enum 表示枚举值
//  <enum value="male" summary="男性" />
//  <enum value="female"><description type="html"><p>女性</p></description></enum>
type Enum struct {
	Deprecated  Semver   `xml:"deprecated,attr,omitempty"`
	Value       string   `xml:"value,attr"`
	Summary     string   `xml:"summary,attr,omitempty"`
	Description Richtext `xml:"description,omitempty"`
}

type shadowEnum Enum

// UnmarshalXML xml.Unmarshaler
func (e *Enum) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	field := "/" + start.Name.Local
	shadow := (*shadowEnum)(e)
	if err := d.DecodeElement(shadow, &start); err != nil {
		return fixedSyntaxError(core.Location{}, err, field)
	}

	if shadow.Value == "" {
		return newSyntaxError(core.Location{}, field+"/@value", locale.ErrRequired)
	}

	if shadow.Description.Text == "" && shadow.Summary == "" {
		return newSyntaxError(core.Location{}, field+"/@summary", locale.ErrRequired)
	}

	return nil
}
