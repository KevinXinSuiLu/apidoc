// SPDX-License-Identifier: MIT

package token

import (
	"io"
	"unicode"

	"golang.org/x/text/message"

	"github.com/caixw/apidoc/v7/core"
	"github.com/caixw/apidoc/v7/internal/lexer"
	"github.com/caixw/apidoc/v7/internal/locale"
)

const (
	cdataStart  = "<![CDATA["
	cdataEnd    = "]]>"
	cdataEscape = "]]]]><![CDATA[>"
)

// Parser 代码块的解析器
type Parser struct {
	*lexer.Lexer
}

// NewParser 声明新的 Parser 实例
func NewParser(b core.Block) (*Parser, error) {
	l, err := lexer.New(b)
	if err != nil {
		return nil, err
	}

	return &Parser{
		Lexer: l,
	}, nil
}

// Token 返回下一个 token 对象
//
// token 可能的类型为 *StartElement、*EndElement、*Instruction、*Attribute、*CData、*Comment 和 *String。
// 其中 *String 用于表示 XML 元素的内容。
//
// 当返回 nil, io.EOF 时，表示已经结束
func (p *Parser) Token() (interface{}, core.Range, error) {
	for {
		if p.AtEOF() {
			return nil, core.Range{}, io.EOF
		}

		pos := p.Position() // 记录元素的开始位置

		bs := p.Next(1)
		if len(bs) == 0 {
			return nil, core.Range{}, io.EOF
		}
		if len(bs) > 1 || bs[0] != '<' {
			p.Rollback() // 当前字符是内容的一部分，返回给 parseContent 解析
			return p.parseContent()
		}

		switch {
		case p.Match("?"):
			return p.parseInstruction(pos)
		case p.Match("![CDATA["):
			return p.parseCData(pos)
		case p.Match("/"):
			return p.parseEndElement(pos)
		case p.Match("!--"):
			return p.parseComment(pos)
		default:
			return p.parseStartElement(pos)
		}
	}
}

func (p *Parser) parseContent() (*String, core.Range, error) {
	start := p.Position()

	data, found := p.Delim('<', false)
	if !found {
		data = p.All()
		if len(data) == 0 {
			return nil, core.Range{}, io.EOF
		}
	}

	r := core.Range{Start: start.Position, End: p.Position().Position}
	return &String{
		Value: string(data),
		Range: r,
	}, r, nil
}

func (p *Parser) parseComment(pos lexer.Position) (*Comment, core.Range, error) {
	start := p.Position()

	data, found := p.DelimString("-->", false)
	if !found {
		return nil, core.Range{}, p.NewError(p.Position().Position, p.Position().Position, "<!--", locale.ErrNotFoundEndTag)
	}
	end := p.Position()
	p.Next(3) // 跳过 --> 三个字符

	r := core.Range{Start: pos.Position, End: p.Position().Position}
	return &Comment{
		Range: r,
		Value: String{
			Range: core.Range{Start: start.Position, End: end.Position},
			Value: string(data),
		},
	}, r, nil
}

func (p *Parser) parseStartElement(pos lexer.Position) (*StartElement, core.Range, error) {
	p.Spaces(0) // 跳过空格

	start := p.Position()
	name, found := p.DelimFunc(func(r rune) bool { return unicode.IsSpace(r) || r == '/' || r == '>' }, false)
	if !found || len(name) == 0 {
		return nil, core.Range{}, p.NewError(p.Position().Position, p.Position().Position, "", locale.ErrInvalidXML)
	}

	elem := &StartElement{
		Name: String{
			Range: core.Range{Start: start.Position, End: p.Position().Position},
			Value: string(name),
		},
	}

	attrs, err := p.parseAttributes()
	if err != nil {
		return nil, core.Range{}, err
	}
	elem.Attributes = attrs

	if p.Match("/>") {
		elem.Range = core.Range{Start: pos.Position, End: p.Position().Position}
		elem.Close = true
		return elem, elem.Range, nil
	}
	if p.Match(">") {
		elem.Range = core.Range{Start: pos.Position, End: p.Position().Position}
		return elem, elem.Range, nil
	}

	return nil, core.Range{}, p.NewError(p.Position().Position, p.Position().Position, string(name), locale.ErrNotFoundEndTag)
}

// pos 表示当前元素的起始位置，包含了 < 元素
func (p *Parser) parseEndElement(pos lexer.Position) (*EndElement, core.Range, error) {
	// 名称开始的定位，传递过来的 pos 表示的 < 起始位置
	start := p.Position()

	name, found := p.Delim('>', false)
	if !found || len(name) == 0 {
		return nil, core.Range{}, p.NewError(p.Position().Position, p.Position().Position, "", locale.ErrInvalidXML)
	}
	end := p.Position()
	p.Next(1) // 去掉 > 符号

	r := core.Range{Start: pos.Position, End: p.Position().Position}
	return &EndElement{
		Range: r,
		Name: String{
			Range: core.Range{Start: start.Position, End: end.Position},
			Value: string(name),
		},
	}, r, nil
}

// pos 表示当前元素的起始位置，包含了 < 元素
func (p *Parser) parseCData(pos lexer.Position) (*CData, core.Range, error) {
	start := p.Position()
	var value []byte

	for {
		v, found := p.DelimString(cdataEnd, false)
		if !found {
			return nil, core.Range{}, p.NewError(pos.Position, p.Position().Position, cdataStart, locale.ErrNotFoundEndTag)
		}
		value = append(value, v...)

		curr := p.Position()
		p.Move(curr.SubRune(']').SubRune(']')) // 回滚两个字符，用于匹配转义内容
		if p.Match(cdataEscape) {
			value = append(value, cdataEscape[2:]...)
			continue
		}

		p.Move(curr)
		break
	}

	end := p.Position()
	p.Next(len(cdataEnd)) // 将 ]]> 从流中去掉

	r := core.Range{Start: pos.Position, End: p.Position().Position}
	return &CData{
		BaseTag: BaseTag{
			Base: Base{Range: r},
			StartTag: String{
				Value: cdataStart,
				Range: core.Range{
					Start: pos.Position,
					End:   core.Position{Line: pos.Line, Character: pos.Character + len(cdataStart)},
				},
			},
			EndTag: String{
				Value: cdataEnd,
				Range: core.Range{
					Start: end.Position,
					End:   core.Position{Line: end.Line, Character: end.Character + len(cdataEnd)},
				},
			},
		},
		Value: String{
			Range: core.Range{Start: start.Position, End: end.Position},
			Value: string(value),
		},
	}, r, nil
}

// pos 表示当前元素的起始位置，包含了 < 元素
func (p *Parser) parseInstruction(pos lexer.Position) (*Instruction, core.Range, error) {
	name, nameRange := p.getName()
	if len(name) == 0 {
		return nil, core.Range{}, p.NewError(p.Position().Position, p.Position().Position, "", locale.ErrInvalidXML)
	}
	elem := &Instruction{
		Name: String{
			Range: nameRange,
			Value: string(name),
		},
	}

	attrs, err := p.parseAttributes()
	if err != nil {
		return nil, core.Range{}, err
	}
	elem.Attributes = attrs

	if p.Match("?>") {
		elem.Range = core.Range{Start: pos.Position, End: p.Position().Position}
		return elem, elem.Range, nil
	}

	return nil, core.Range{}, p.NewError(p.Position().Position, p.Position().Position, "<?", locale.ErrNotFoundEndTag)
}

func (p *Parser) parseAttributes() (attrs []*Attribute, err error) {
	for {
		attr, err := p.parseAttribute()
		if err != nil {
			return nil, err
		}
		if attr == nil {
			break
		}

		attrs = append(attrs, attr)
	}
	p.Spaces(0)

	return attrs, nil
}

func (p *Parser) parseAttribute() (*Attribute, error) {
	p.Spaces(0) // 忽略空格
	pos, start := p.Position(), p.Position()

	name, nameRange := p.getName()
	if len(name) == 0 {
		return nil, nil
	}
	attr := &Attribute{
		Name: String{
			Range: nameRange,
			Value: string(name),
		},
	}

	p.Spaces(0)
	if !p.Match("=") {
		return nil, p.NewError(p.Position().Position, p.Position().Position, "", locale.ErrInvalidXML)
	}

	p.Spaces(0)
	if !p.Match("\"") {
		return nil, p.NewError(p.Position().Position, p.Position().Position, "", locale.ErrInvalidXML)
	}

	pos = p.Position()
	value, found := p.Delim('"', true)
	if !found || len(value) == 0 {
		return nil, p.NewError(p.Position().Position, p.Position().Position, "", locale.ErrInvalidXML)
	}
	end := p.Position().SubRune('"') // 不包含 " 符号
	attr.Value = String{
		Range: core.Range{Start: pos.Position, End: end.Position},
		Value: string(p.Bytes(pos.Offset, end.Offset)),
	}

	attr.Range = core.Range{Start: start.Position, End: p.Position().Position}

	return attr, nil
}

func (p *Parser) getName() ([]byte, core.Range) {
	start := p.Position()

	for {
		if p.AtEOF() {
			break
		}

		rs := p.Next(1)
		if len(rs) != 1 {
			continue
		}

		b := rs[0]
		if b == '"' || b == '=' || b == '<' || b == '>' || b == '?' || b == '/' || unicode.IsSpace(rune(b)) {
			p.Rollback()
			break
		}
	}

	end := p.Position()
	return p.Bytes(start.Offset, end.Offset), core.Range{Start: start.Position, End: end.Position}
}

// NewError 生成 *core.SyntaxError 对象
func (p *Parser) NewError(start, end core.Position, field string, key message.Reference, v ...interface{}) error {
	return core.NewSyntaxError(core.Location{
		URI:   p.Location.URI,
		Range: core.Range{Start: start, End: end},
	}, field, key, v...)
}

// WithError 将 err 包装成 *core.SyntaxError 类型
//
// 如果 err 本身就是 *core.SyntaxError 类型，则只取 err.Err
// 作为返回对象有的 Err 字段，其它字段弃用。
func (p *Parser) WithError(start, end core.Position, field string, err error) error {
	if serr, ok := err.(*core.SyntaxError); ok {
		err = serr.Err
	}

	return core.NewSyntaxErrorWithError(core.Location{
		URI:   p.Location.URI,
		Range: core.Range{Start: start, End: end},
	}, field, err)
}