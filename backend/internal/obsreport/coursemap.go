package obsreport

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"velocloud-academy/backend/internal/content"
)

// Goal mirrors frontend/src/progress/goals.ts (coursemap_test.go fails when the two differ).
type Goal struct {
	ID, Title string
	Sections  []string
	Labs      []string
	Screens   []string
}

// Goals are the four things the course is for.
var Goals = []Goal{
	{"read-config", "Read an Edge's configuration", []string{"configuration", "architecture"},
		[]string{"video-wrong-link", "capstone-3-change-request", "capstone-1-monday-morning"},
		[]string{"Edge list and Edge tabs (Device, Business Policy, Firewall)", "What-if", "Challenge", "Flow tracer (policy, firewall stages)"}},
	{"traffic-flow", "How traffic flows", []string{"dmpo", "configuration", "architecture"},
		[]string{"video-wrong-link", "mtu-ipsec", "capstone-1-monday-morning", "capstone-2-hub-incident"},
		[]string{"Flow tracer (all eight stages)", "Path explainer", "Monitor > Links and Paths", "Edge Business Policy tab"}},
	{"tunnels", "How tunnels form", []string{"architecture", "overlay"},
		[]string{"no-direct-tunnel", "hub-down", "capstone-2-hub-incident"},
		[]string{"Map", "Monitor > Paths", "Path explainer", "Timeline", "Overlay challenge"}},
	{"routes", "How routes flow", []string{"routing"},
		[]string{"branch-b-cannot-reach", "which-route-wins", "capstone-1-monday-morning"},
		[]string{"Routes (Route Table Dump layout)", "Reachability explainer", "Routing challenge", "Flow tracer (route stage)"}},
}

var sectionScreens = map[string][]string{
	"introduction":  {"(none required; the simulator picker is optional)"},
	"architecture":  {"Edge list and tabs", "Map", "Monitor > Links", "Flow tracer (MTU stage)", "Path explainer (MTU)"},
	"dmpo":          {"Monitor > Links and Paths", "Path explainer (link steering)", "Flow tracer (path and remediation stages)", "Edge Business Policy tab"},
	"configuration": {"Edge Device, Business Policy and Firewall tabs", "What-if", "Challenge", "Flow tracer (policy and firewall stages)"},
	"overlay":       {"Map", "Monitor > Edges (Links, Paths)", "Path explainer", "Timeline", "Overlay challenge"},
	"routing":       {"Routes", "Reachability explainer", "Routing challenge", "Flow tracer (route stage)"},
}

var diagramRe = regexp.MustCompile(`<Diagram id="([a-z0-9-]+)"`)

// WeakSpots is the author's reading of where a goal is thinly covered and what would fix it (nothing here is built).
var weakSpots = map[string][]string{
	"read-config": {
		"Device-tab settings are modelled as opaque values (assumption 33): only eight settings exist, so the learner cannot practise reading real interface, VLAN or WAN-link settings. Would fix it: an observation of the real Device tab (which settings, which are overridable) and then more settings in the registry.",
		"There is no lab for Firewall rules or Segments; labs touch Business Policy and device overrides only. Would fix it: a lab on a Firewall rule that is overridden at an Edge, once firewall rule order and default action (VERIFY) are observed.",
		"How the Edge editor marks an override is a VERIFY row, so the debrief says \"unverified path\" there. Would fix it: one lab observation.",
	},
	"traffic-flow": {
		"Remediation (FEC, jitter buffer, NACK) is described but whether it is active for a flow is not modelled (VERIFY), so no lab asks about a brownout. Would fix it: an observation of how the VCO shows active remediation, then a brownout lab.",
		"The business-policy default when no rule matches and the order of firewall versus business policy are VERIFY rows; the flow tracer shows them as assumptions 2, 85, 86 and 91, and no lab can rest on them. Would fix it: observations.",
		"No lab covers QoS, DSCP or Service Class marking. Would fix it: a lab on a rule's Service Class once the Applications screen path is known (VERIFY).",
	},
	"tunnels": {
		"Non SD-WAN Destinations (NSD) have a lesson, a diagram and a map entry but no lab. Would fix it: a lab on an NSD tunnel that is down, using only the documented states.",
		"Behaviour when the Orchestrator is unreachable and inbound port needs of an Edge acting as a Hub are VERIFY, so the tunnel labs avoid them. Would fix it: observations.",
		"Hub clusters are modelled in the demo topology but no lab uses them. Would fix it: a cluster failover lab once the Cluster Score location (VERIFY) is known.",
	},
	"routes": {
		"The Gateway as route reflector is not modelled (VERIFY), so a route that only the Gateway can explain cannot be practised. Would fix it: an observation of the Gateway Route Table.",
		"The redistribution defaults, the BGP filter default action and OE1/OE2 preference numbers are assumptions or VERIFY rows; the which-route-wins lab rests on four assumptions. Would fix it: observations of the five redistribution check boxes and of List Routes per Prefix.",
		"No lab covers a static route with an ICMP probe or a default route, and BFD is only mentioned in passing (a BGP option and an example of an overridable setting), with no lesson of its own. Would fix it: a static-route lab (the probe is already modelled) and a BFD lesson once sourced.",
	},
}

// CourseMapMarkdown renders docs/COURSE-MAP.md from the real content.
func CourseMapMarkdown(contentDir string) (string, error) {
	c, err := content.Load(os.DirFS(contentDir))
	if err != nil {
		return "", fmt.Errorf("content is invalid, fix it first: %w", err)
	}
	lessonSection := map[string]string{}
	for _, s := range c.Sections {
		for _, l := range s.Lessons {
			lessonSection[l.Meta.ID] = s.Meta.ID
		}
	}
	labsBySection := map[string][]string{}
	for _, lab := range c.Labs() {
		if strings.HasPrefix(lab.Lab.ID, "self-check-") {
			continue
		}
		seen := map[string]bool{}
		for _, g := range lab.Lab.Goals {
			if sid := lessonSection[g]; sid != "" && !seen[sid] {
				seen[sid] = true
				labsBySection[sid] = append(labsBySection[sid], lab.Lab.ID)
			}
		}
	}
	var b strings.Builder
	b.WriteString("# Course map\n\n_Generated by `make course-map` from the content; do not edit by hand. The goal table mirrors `frontend/src/progress/goals.ts`._\n\n")
	b.WriteString("What the course is for: **read an Edge's configuration**, **know how traffic flows**, **know how tunnels form**, **know how routes flow**. Each section below lists what teaches it and which goal(s) it supports.\n\n")
	secGoals := map[string][]string{}
	for _, g := range Goals {
		for _, s := range g.Sections {
			secGoals[s] = append(secGoals[s], g.Title)
		}
	}
	for _, s := range c.Sections {
		fmt.Fprintf(&b, "## %s (`%s`)\n\n%s\n\n", s.Meta.Title, s.Meta.ID, strings.TrimSpace(s.Meta.Summary))
		gs := secGoals[s.Meta.ID]
		if len(gs) == 0 {
			gs = []string{"foundation for all four goals (no single goal)"}
		}
		fmt.Fprintf(&b, "- **Supports:** %s\n", strings.Join(gs, "; "))
		var lessons []string
		diagrams := map[string]bool{}
		for _, l := range s.Lessons {
			lessons = append(lessons, fmt.Sprintf("%s (%d min)", l.Meta.ID, l.Meta.EstimatedMinutes))
			for _, m := range diagramRe.FindAllStringSubmatch(l.Body, -1) {
				diagrams[m[1]] = true
			}
		}
		fmt.Fprintf(&b, "- **Lessons (%d):** %s\n", len(s.Lessons), strings.Join(lessons, ", "))
		topics := map[string]int{}
		for _, q := range s.Quiz {
			topics[q.Topic]++
		}
		var tl []string
		for t, n := range topics {
			tl = append(tl, fmt.Sprintf("%s (%d)", t, n))
		}
		sort.Strings(tl)
		fmt.Fprintf(&b, "- **Quiz (%d questions), topics:** %s\n", len(s.Quiz), strings.Join(tl, ", "))
		var dl []string
		for d := range diagrams {
			dl = append(dl, d)
		}
		sort.Strings(dl)
		fmt.Fprintf(&b, "- **Diagrams (%d):** %s\n", len(dl), orDash(strings.Join(dl, ", ")))
		labs := labsBySection[s.Meta.ID]
		sort.Strings(labs)
		fmt.Fprintf(&b, "- **Labs that reinforce it:** %s\n", orDash(strings.Join(labs, ", ")))
		fmt.Fprintf(&b, "- **Simulator screens:** %s\n\n", strings.Join(sectionScreens[s.Meta.ID], "; "))
	}
	b.WriteString("## Goals and their coverage\n\n| goal | sections | lessons | quiz questions | diagrams | labs | simulator screens |\n|---|---|---|---|---|---|---|\n")
	type cov struct{ lessons, questions, diagrams, labs int }
	covs := map[string]cov{}
	for _, g := range Goals {
		var cv cov
		d := map[string]bool{}
		for _, sid := range g.Sections {
			if s, ok := c.Section(sid); ok {
				cv.lessons += len(s.Lessons)
				cv.questions += len(s.Quiz)
				for _, l := range s.Lessons {
					for _, m := range diagramRe.FindAllStringSubmatch(l.Body, -1) {
						d[m[1]] = true
					}
				}
			}
		}
		cv.diagrams, cv.labs = len(d), len(g.Labs)
		covs[g.ID] = cv
		fmt.Fprintf(&b, "| %s | %s | %d | %d | %d | %s | %s |\n", g.Title, strings.Join(g.Sections, ", "), cv.lessons, cv.questions, cv.diagrams, strings.Join(g.Labs, ", "), strings.Join(g.Screens, "; "))
	}
	b.WriteString("\nSections that support more than one goal are counted in each (architecture and configuration), so the totals overlap. The capstone labs and the self-check (`/sim/self-check`) span all four goals.\n\n")
	writeStudyGuide(&b, c)
	b.WriteString("## Weak coverage and what would fix it (not built)\n\n")
	b.WriteString("A goal is weak where the course can describe something but the learner cannot practise it honestly, usually because a fact is VERIFY. The cure is mostly an observation (see `docs/HOW-TO-ADD-AN-OBSERVATION.md` and `docs/OBSERVATION-PRIORITIES.md`), then a lab.\n\n")
	for _, g := range Goals {
		fmt.Fprintf(&b, "### %s\n\n", g.Title)
		for _, w := range weakSpots[g.ID] {
			fmt.Fprintf(&b, "- %s\n", w)
		}
		b.WriteString("\n")
	}
	b.WriteString("### Across the whole course\n\n")
	b.WriteString("- **Order of sections:** DMPO (section 3) uses business-policy rules and link steering before Configuration (section 4) teaches them; the DMPO steering lesson defines what it needs and the lessons carry `Term` popovers, but a learner who skips ahead to Configuration first loses nothing. Section 4's firewall and QoS lessons mention OSPF before section 6; they are previews with popovers.\n")
	b.WriteString("- **Observation-limited:** most of the weakness above disappears when VERIFY rows become OBSERVED; `docs/OBSERVATION-PRIORITIES.md` ranks them.\n")
	return b.String(), nil
}

// writeStudyGuide adds the study guide mapping (content/study-guide.yaml): per topic its coverage, lessons and labs, and the
// counts of topics by coverage level. Nothing here is hand-written.
func writeStudyGuide(b *strings.Builder, c *content.Course) {
	g := c.StudyGuide()
	if g == nil {
		return
	}
	b.WriteString("## Study guide mapping\n\n")
	b.WriteString("The learner's six-section study guide (navigation labels only) and what the course offers for each topic, from `content/study-guide.yaml`. Coverage: **full** = a lesson teaches the topic; **partial** = a lesson teaches part of it, or only the concept and where it lives in the VCO; **none** = nothing in the course teaches it. The dashboard counts progress over covered topics only.\n\n")
	counts := map[string]int{}
	for _, s := range g.Sections {
		for _, t := range s.Topics {
			counts[t.Coverage]++
		}
	}
	fmt.Fprintf(b, "Topics: %d full, %d partial, %d none.\n\n", counts[content.CoverageFull], counts[content.CoveragePartial], counts[content.CoverageNone])
	for _, s := range g.Sections {
		fmt.Fprintf(b, "### %s: %s\n\n| topic | kind | coverage | lessons | labs |\n|---|---|---|---|---|\n", s.ID, s.Title)
		for _, t := range s.Topics {
			fmt.Fprintf(b, "| %s | %s | %s | %s | %s |\n", t.Title, t.Kind, t.Coverage, orDash(strings.Join(t.Lessons, ", ")), orDash(strings.Join(t.Labs, ", ")))
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(b, "Lessons deliberately outside the guide: %s.\n\n", orDash(strings.Join(g.UnmappedLessons, ", ")))
	for _, w := range c.GuideWarnings() {
		fmt.Fprintf(b, "- Warning: %s\n", w.Message)
	}
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
