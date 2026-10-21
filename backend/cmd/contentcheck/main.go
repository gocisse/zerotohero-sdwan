// Command contentcheck validates the course content without starting the server.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"velocloud-academy/backend/internal/content"
)

func main() {
	dir := flag.String("dir", "../content", "path to the content directory")
	model := flag.String("model", "../docs/SIMULATOR-MODEL.md", "docs/SIMULATOR-MODEL.md, used to check that observations name a real assumption number (skipped if the file is missing)")
	flag.Parse()

	c, err := content.Load(os.DirFS(*dir))
	// Observations are also checked against the assumption numbers of docs/SIMULATOR-MODEL.md, which the server container does
	// not have; those findings are reported together with the load errors.
	var modelProbs []content.ObsProblem
	if data, rerr := os.ReadFile(*model); rerr == nil {
		for _, p := range content.CheckObservations(content.ReadReferences(os.DirFS(*dir)), content.ModelAssumptions(string(data))) {
			if strings.Contains(p.Message, "SIMULATOR-MODEL") {
				modelProbs = append(modelProbs, p)
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "note: %s not found; observation assumption numbers were not checked\n", *model)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		for _, p := range modelProbs {
			fmt.Fprintf(os.Stderr, "  %s: %s\n", p.File, p.Message)
		}
		os.Exit(1)
	}
	if len(modelProbs) > 0 {
		for _, p := range modelProbs {
			fmt.Fprintf(os.Stderr, "%s: %s\n", p.File, p.Message)
		}
		os.Exit(1)
	}
	lessons, questions := 0, 0
	for _, s := range c.Sections {
		lessons += len(s.Lessons)
		questions += len(s.Quiz)
	}
	fmt.Printf("content OK: %d section(s), %d lesson(s), %d quiz question(s), %d diagram(s), %d glossary term(s), %d topology file(s), %d lab(s), %d runbook(s)\n", len(c.Sections), lessons, questions, c.DiagramCount(), len(c.Glossary), len(c.Topologies()), len(c.Labs()), len(c.Runbooks()))
	for _, w := range c.GuideWarnings() {
		fmt.Printf("warning: %s: %s\n", w.File, w.Message)
	}
	for _, t := range c.Topologies() {
		for _, w := range t.Warnings {
			fmt.Printf("warning: %s: %s: %s\n", t.Path, w.Path, w.Message)
		}
	}
}
