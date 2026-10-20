// Command contentcheck validates the course content without starting the server.
package main

import (
	"flag"
	"fmt"
	"os"

	"velocloud-academy/backend/internal/content"
)

func main() {
	dir := flag.String("dir", "../content", "path to the content directory")
	flag.Parse()

	c, err := content.Load(os.DirFS(*dir))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	lessons, questions := 0, 0
	for _, s := range c.Sections {
		lessons += len(s.Lessons)
		questions += len(s.Quiz)
	}
	fmt.Printf("content OK: %d section(s), %d lesson(s), %d quiz question(s), %d diagram(s), %d glossary term(s), %d topology file(s)\n", len(c.Sections), lessons, questions, c.DiagramCount(), len(c.Glossary), len(c.Topologies()))
	for _, t := range c.Topologies() {
		for _, w := range t.Warnings {
			fmt.Printf("warning: %s: %s: %s\n", t.Path, w.Path, w.Message)
		}
	}
}
