package parser

import (
	"context"
	"io"
	"strings"

	"github.com/elijahthis/baby-crawler/internal/shared"
	"golang.org/x/net/html"
)

type HTMLParser struct{}

func NewHTMLParser() *HTMLParser {
	return &HTMLParser{}
}

func (p *HTMLParser) Parse(ctx context.Context, r io.Reader) (shared.ParsedData, error) {
	data := shared.ParsedData{}
	tokenizer := html.NewTokenizer(r)

	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			if tokenizer.Err() == io.EOF {
				return data, nil
			}
			return data, tokenizer.Err()
		}

		token := tokenizer.Token()
		switch tokenType {
		case html.StartTagToken:
			if token.Data == "a" {
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						data.Links = append(data.Links, attr.Val)
					}
				}
			}
		case html.TextToken:
			data.Text += strings.TrimSpace(token.Data) + " "
		}
	}
}
