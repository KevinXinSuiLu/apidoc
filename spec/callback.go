// SPDX-License-Identifier: MIT

package spec

import (
	"encoding/xml"

	"github.com/caixw/apidoc/v7/core"
	"github.com/caixw/apidoc/v7/internal/locale"
)

// Callback 回调函数的定义
//  <Callback deprecated="1.1.1" method="GET">
//       <request status="200" mimetype="json" type="object">
//           <param name="name" type="string" />
//           <param name="sex" type="string">
//               <enum value="male" summary="male" />
//               <enum value="female" summary="female" />
//           </param>
//           <param name="age" type="number" />
//       </request>
//   </Callback>
type Callback struct {
	Method      Method     `xml:"method,attr"`
	Path        *Path      `xml:"path,omitempty"`
	Summary     string     `xml:"summary,attr,omitempty"`
	Description Richtext   `xml:"description,omitempty"`
	Deprecated  Semver     `xml:"deprecated,attr,omitempty"`
	Reference   string     `xml:"ref,attr,omitempty"`
	Responses   []*Request `xml:"response,omitempty"`
	Requests    []*Request `xml:"request"` // 至少一个
	Headers     []*Param   `xml:"header,omitempty"`
}

type shadowCallback Callback

// UnmarshalXML xml.Unmarshaler
func (c *Callback) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	field := "/" + start.Name.Local

	shadow := (*shadowCallback)(c)
	if err := d.DecodeElement(shadow, &start); err != nil {
		return fixedSyntaxError(core.Location{}, err, field)
	}

	if shadow.Method == "" {
		return newSyntaxError(core.Location{}, field+"/@method", locale.ErrRequired)
	}

	if len(shadow.Requests) == 0 {
		return newSyntaxError(core.Location{}, field+"/request", locale.ErrRequired)
	}

	// 可以不需要 response

	return nil
}
