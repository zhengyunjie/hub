package md2roff

import (
	"bytes"
	"fmt"
	"io"
	"regexp"

	"github.com/russross/blackfriday"
)

var (
	backslash     = []byte{'\\'}
	htmlEscape    = regexp.MustCompile(`<([A-Za-z][A-Za-z0-9_-]*)>`)
	roffEscape    = regexp.MustCompile(`[&\~_-]`)
	headingEscape = regexp.MustCompile(`["]`)
	titleRe       = regexp.MustCompile(`(?P<name>[A-Za-z][A-Za-z0-9_-]+)\((?P<num>\d)\) -- (?P<title>.+)`)
)

func escape(src []byte, re *regexp.Regexp) []byte {
	return re.ReplaceAllFunc(src, func(c []byte) []byte {
		return append(backslash, c...)
	})
}

type RoffRenderer struct {
	Version string
	Date    string

	itemIndex int
}

func (r *RoffRenderer) RenderHeader(buf io.Writer, ast *blackfriday.Node) {
}

func (r *RoffRenderer) RenderFooter(buf io.Writer, ast *blackfriday.Node) {
}

func (r *RoffRenderer) RenderNode(buf io.Writer, node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
	if entering {
		switch node.Type {
		case blackfriday.Emph:
			io.WriteString(buf, `\fI`)
		case blackfriday.Strong:
			io.WriteString(buf, `\fB`)
		case blackfriday.Link:
			io.WriteString(buf, `\[la]`)
		case blackfriday.Code:
			io.WriteString(buf, `\fB\fC`)
		case blackfriday.Hardbreak:
			io.WriteString(buf, "\n.br\n")
		case blackfriday.Paragraph:
			if node.Parent.Type != blackfriday.Item {
				io.WriteString(buf, ".PP\n")
			}
		case blackfriday.CodeBlock:
			io.WriteString(buf, ".PP\n.RS\n.nf\n")
		case blackfriday.List:
			r.itemIndex = 0
		case blackfriday.Item:
			if r.itemIndex%2 == 0 {
				io.WriteString(buf, ".PP\n")
			} else {
				io.WriteString(buf, ".RS 4\n")
			}
			r.itemIndex += 1
		case blackfriday.Heading:
			renderHeading(buf, node, r.Date, r.Version)
			return blackfriday.SkipChildren
		}
	}

	leaf := len(node.Literal) > 0
	if leaf {
		buf.Write(escape(node.Literal, roffEscape))
	}

	if !entering || leaf {
		switch node.Type {
		case blackfriday.Emph,
			blackfriday.Strong:
			io.WriteString(buf, `\fP`)
		case blackfriday.Link:
			io.WriteString(buf, `\[ra]`)
		case blackfriday.Code:
			io.WriteString(buf, `\fR`)
		case blackfriday.CodeBlock:
			io.WriteString(buf, "\n.fi\n.RE\n")
		case blackfriday.HTMLSpan,
			blackfriday.Del,
			blackfriday.Image:
		case blackfriday.Item:
			if r.itemIndex%2 == 0 {
				io.WriteString(buf, ".RE\n")
			}
		default:
			if !leaf {
				io.WriteString(buf, "\n")
			}
		}
	}

	return blackfriday.GoToNext
}

func textContent(node *blackfriday.Node) []byte {
	var buf bytes.Buffer
	node.Walk(func(n *blackfriday.Node, entering bool) blackfriday.WalkStatus {
		if entering && len(n.Literal) > 0 {
			buf.Write(n.Literal)
		}
		return blackfriday.GoToNext
	})
	return buf.Bytes()
}

func renderHeading(buf io.Writer, node *blackfriday.Node, date, version string) {
	text := textContent(node)
	switch node.HeadingData.Level {
	case 1:
		var name []byte
		var num []byte
		if match := titleRe.FindAllSubmatch(text, 1); match != nil {
			name, num, text = match[0][1], match[0][2], match[0][3]
		}
		fmt.Fprintf(buf, ".TH \"%s\" \"%s\" \"%s\" \"%s\" \"%s\"\n",
			escape(name, headingEscape),
			num,
			escape([]byte(date), headingEscape),
			escape([]byte(version), headingEscape),
			escape(text, headingEscape),
		)
		io.WriteString(buf, ".nh\n")   // disable hyphenation
		io.WriteString(buf, ".ad l\n") // disable justification
	case 2, 3:
		var ht string
		switch node.HeadingData.Level {
		case 2:
			ht = ".SH"
		case 3:
			ht = ".SS"
		}
		fmt.Fprintf(buf, "%s \"%s\"\n", ht, escape(text, headingEscape))
	}
}

func sanitizeInput(src []byte) []byte {
	return htmlEscape.ReplaceAllFunc(src, func(match []byte) []byte {
		openBracket := []byte(`\<`)
		closeBracket := []byte(`\>`)
		res := append(openBracket, match[1:len(match)-1]...)
		return append(res, closeBracket...)
	})
}

type renderOption struct {
	renderer blackfriday.Renderer
	buffer   io.Writer
}

func Opt(buffer io.Writer, renderer blackfriday.Renderer) *renderOption {
	return &renderOption{renderer, buffer}
}

func Generate(src []byte, opts ...*renderOption) {
	parser := blackfriday.New(blackfriday.WithExtensions(blackfriday.CommonExtensions))
	ast := parser.Parse(sanitizeInput(src))

	for _, opt := range opts {
		opt.renderer.RenderHeader(opt.buffer, ast)
		ast.Walk(func(node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
			return opt.renderer.RenderNode(opt.buffer, node, entering)
		})
		opt.renderer.RenderFooter(opt.buffer, ast)
	}
}
