package main

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/github/hub/md2roff"
	"github.com/russross/blackfriday"
)

func generateFromFile(mdFile string) error {
	content, err := ioutil.ReadFile(mdFile)
	if err != nil {
		return err
	}

	roffFile := strings.TrimSuffix(mdFile, ".md") + ".1"
	roffBuf, err := os.OpenFile(roffFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer roffBuf.Close()

	htmlFile := strings.TrimSuffix(mdFile, ".md") + ".html"
	htmlBuf, err := os.OpenFile(htmlFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer htmlBuf.Close()

	html := blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{
		Flags: blackfriday.HTMLFlagsNone,
	})
	roff := &md2roff.RoffRenderer{
		Version: "Hub v2.4.0",
		Date:    "Jan 5, 1984",
	}

	md2roff.Generate(content,
		md2roff.Opt(roffBuf, roff),
		md2roff.Opt(htmlBuf, html),
	)

	return nil
}

func main() {
	for _, infile := range os.Args[1:] {
		err := generateFromFile(infile)
		if err != nil {
			panic(err)
		}
	}
}
