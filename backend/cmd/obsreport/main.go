// Command obsreport writes docs/OBSERVATION-PRIORITIES.md (make observations-report).
package main

import (
	"flag"
	"fmt"
	"os"

	"velocloud-academy/backend/internal/obsreport"
)

func main() {
	contentDir := flag.String("content", "../content", "content directory")
	simDir := flag.String("sim", "internal/sim", "engine package directory (for citation counts)")
	model := flag.String("model", "../docs/SIMULATOR-MODEL.md", "docs/SIMULATOR-MODEL.md")
	out := flag.String("out", "../docs/OBSERVATION-PRIORITIES.md", "output file")
	courseMap := flag.String("course-map", "", "write docs/COURSE-MAP.md to this file instead of the observation report")
	flag.Parse()
	if *courseMap != "" {
		md, err := obsreport.CourseMapMarkdown(*contentDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := os.WriteFile(*courseMap, []byte(md), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s\n", *courseMap)
		return
	}
	data, err := os.ReadFile(*model)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	rep, err := obsreport.Build(*contentDir, *simDir, string(data))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, []byte(rep.Markdown()), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s: %d VERIFY row(s), %d assumption(s), %d item(s) with an observation\n", *out, rep.VerifyRows, rep.Assumptions, rep.ResolvedCount)
}
