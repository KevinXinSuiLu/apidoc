// SPDX-License-Identifier: MIT

// Package apidoc RESTful API 文档生成工具。
package apidoc

import (
	"bytes"
	"context"
	"sync"

	"golang.org/x/text/language"

	"github.com/caixw/apidoc/v5/doc"
	i "github.com/caixw/apidoc/v5/internal/input"
	"github.com/caixw/apidoc/v5/internal/locale"
	o "github.com/caixw/apidoc/v5/internal/output"
	"github.com/caixw/apidoc/v5/internal/vars"
	"github.com/caixw/apidoc/v5/message"
	"github.com/caixw/apidoc/v5/options"
)

// InitLocale 初始化语言环境
//
// NOTE: 必须保证在第一时间调用。
//
// 如果 tag 的值为 language.Und，则表示采用系统语言
func InitLocale(tag language.Tag) error {
	return locale.Init(tag)
}

// Version 获取当前程序的版本号
func Version() string {
	return vars.Version()
}

// Do 分析输入信息，并最终输出到指定的文件。
//
// h 表示处理语法错误的处理器。
// output 输出设置项；
// inputs 输入设置项。
func Do(ctx context.Context, h *message.Handler, output *options.Output, inputs ...*options.Input) error {
	doc, err := Parse(ctx, h, inputs...)
	if err != nil {
		return err
	}

	if err := doc.Sanitize(); err != nil {
		h.Error(err)
	}

	return o.Render(doc, output)
}

// Parse 分析从 block 中获取的代码块。并填充到 Doc 中
//
// 当所有的代码块已经放入 Block 之后，Block 会被关闭。
//
// 所有与解析有关的错误均通过 h 输出。而其它错误，比如参数问题等，通过返回参数返回。
func Parse(ctx context.Context, h *message.Handler, input ...*options.Input) (*doc.Doc, error) {
	block, err := i.Parse(ctx, h, input...)
	if err != nil {
		return nil, err
	}

	doc := doc.New()
	wg := sync.WaitGroup{}

LOOP:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case blk, ok := <-block:
			if !ok {
				break LOOP
			}

			wg.Add(1)
			go func(b i.Block) {
				parseBlock(doc, b, h)
				wg.Done()
			}(blk)
		}
	}

	wg.Wait()

	return doc, nil
}

var (
	apidocBegin = []byte("<apidoc")
	apiBegin    = []byte("<api")
)

func parseBlock(d *doc.Doc, block i.Block, h *message.Handler) {
	var err error
	switch {
	case bytes.HasPrefix(block.Data, apidocBegin):
		err = d.FromXML(block.Data)
	case bytes.HasPrefix(block.Data, apiBegin):
		err = d.NewAPI(block.File, block.Line).FromXML(block.Data)
	}

	h.Error(message.WithError(block.File, "", block.Line, err))
}
