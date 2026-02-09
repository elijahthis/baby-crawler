package main

import (
	"github.com/elijahthis/baby-crawler/cmd"
	"github.com/elijahthis/baby-crawler/internal/shared"
)

func main() {
	shared.InitLogger("crawler")

	cmd.ExecuteCrawler()
}
