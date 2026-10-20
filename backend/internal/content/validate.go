package content

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var idPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Validate applies every cross-file rule to b and returns all violations.
func Validate(b *Bundle) ErrorList {
	v := &validator{b: b}
	v.run()
	return v.errs
}

type validator struct {
	b    *Bundle
	errs ErrorList

	sectionPath map[string]string // section id -> section.yaml path
	lessonPath  map[string]string // lesson id -> mdx path
	diagramPath map[string]string // diagram id -> yaml path
	termIDs     map[string]bool
}

func (v *validator) run() {
	b := v.b
	v.sectionPath = map[string]string{}
	v.lessonPath = map[string]string{}
	v.diagramPath = map[string]string{}
	v.termIDs = map[string]bool{}

	for _, x := range b.Extra {
		v.errs.add(x, "unexpected file or directory (orphan): not part of the content format")
	}

	// Pass 1: collect ids so references can be resolved.
	anyBrokenSection, anyBrokenLesson := false, false
	for _, s := range b.Sections {
		if s.MetaBroken {
			anyBrokenSection = true
		} else if s.Meta.ID != "" {
			if prev, dup := v.sectionPath[s.Meta.ID]; dup {
				v.errs.add(s.MetaPath, "duplicate section id %q (also defined in %s)", s.Meta.ID, prev)
			} else {
				v.sectionPath[s.Meta.ID] = s.MetaPath
			}
		}
		for _, l := range s.Lessons {
			if l.Broken {
				anyBrokenLesson = true
			} else if l.Meta.ID != "" {
				if prev, dup := v.lessonPath[l.Meta.ID]; dup {
					v.errs.add(l.Path, "duplicate lesson id %q (also defined in %s)", l.Meta.ID, prev)
				} else {
					v.lessonPath[l.Meta.ID] = l.Path
				}
			}
		}
	}

	anyBrokenDiagram := false
	for _, d := range b.Diagrams {
		if d.Broken {
			anyBrokenDiagram = true
			continue
		}
		validateDiagram(&v.errs, d.Path, d.D)
		if d.D.ID == "" {
			continue
		}
		if prev, dup := v.diagramPath[d.D.ID]; dup {
			v.errs.add(d.Path, "duplicate diagram id %q (also defined in %s)", d.D.ID, prev)
		} else {
			v.diagramPath[d.D.ID] = d.Path
		}
	}
	if b.HasGlossary && !b.GlossaryBad {
		validateGlossary(&v.errs, b.GlossaryPath, b.Glossary, v.lessonPath, anyBrokenLesson)
		for _, t := range b.Glossary {
			v.termIDs[t.ID] = true
		}
	}
	for _, r := range b.References {
		validateReference(&v.errs, r)
	}
	for _, p := range CheckObservations(b.References, nil) {
		v.errs.add(p.File, "%s", p.Message)
	}
	validateTopologies(&v.errs, b.Topologies)
	validateLabs(&v.errs, b.Labs, b.Topologies, b.References, v.lessonPath, anyBrokenLesson)
	v.componentRefs(anyBrokenDiagram, b.GlossaryBad)
	ge, _ := checkGuide(b)
	v.errs = append(v.errs, ge...)

	if !b.CourseBroken {
		v.course(anyBrokenSection)
	}
	for _, s := range b.Sections {
		v.section(s, anyBrokenLesson)
	}
	v.prerequisiteCycles()
	v.questionIDs()
}

func (v *validator) course(anyBrokenSection bool) {
	b := v.b
	p := b.CoursePath
	if strings.TrimSpace(b.Course.Title) == "" {
		v.errs.add(p, "title is required")
	}
	if strings.TrimSpace(b.Course.Description) == "" {
		v.errs.add(p, "description is required")
	}
	if len(b.Course.Sections) == 0 {
		v.errs.add(p, "sections must list at least one section id")
	}
	seen := map[string]bool{}
	for _, id := range b.Course.Sections {
		if seen[id] {
			v.errs.add(p, "section id %q is listed more than once", id)
			continue
		}
		seen[id] = true
		if _, ok := v.sectionPath[id]; !ok && !anyBrokenSection {
			v.errs.add(p, "section id %q is listed but no section.yaml defines it", id)
		}
	}
	for _, s := range b.Sections {
		if !s.MetaBroken && s.Meta.ID != "" && !seen[s.Meta.ID] {
			v.errs.add(s.MetaPath, "orphan section %q is not listed in course.yaml", s.Meta.ID)
		}
	}
}

func (v *validator) section(s SectionBundle, anyBrokenLesson bool) {
	if !s.MetaBroken {
		p, m := s.MetaPath, s.Meta
		v.requireID(p, "id", m.ID)
		if strings.TrimSpace(m.Title) == "" {
			v.errs.add(p, "title is required")
		}
		if strings.TrimSpace(m.Summary) == "" {
			v.errs.add(p, "summary is required")
		}
		if m.EstimatedMinutes <= 0 {
			v.errs.add(p, "estimated_minutes must be a positive number")
		}
		if len(m.Lessons) == 0 {
			v.errs.add(p, "lessons must list at least one lesson id")
		}

		listed := map[string]bool{}
		inSection := map[string]bool{}
		for _, l := range s.Lessons {
			if !l.Broken {
				inSection[l.Meta.ID] = true
			}
		}
		for _, id := range m.Lessons {
			if listed[id] {
				v.errs.add(p, "lesson id %q is listed more than once", id)
				continue
			}
			listed[id] = true
			if !inSection[id] && !anyBrokenLesson {
				if other, ok := v.lessonPath[id]; ok {
					v.errs.add(p, "lesson id %q is defined in %s, outside this section's lessons/ folder", id, other)
				} else {
					v.errs.add(p, "lesson id %q is listed but no lessons/*.mdx file defines it", id)
				}
			}
		}
		for _, l := range s.Lessons {
			if !l.Broken && l.Meta.ID != "" && !listed[l.Meta.ID] {
				v.errs.add(l.Path, "orphan lesson %q is not listed in %s", l.Meta.ID, p)
			}
		}
	}

	for _, l := range s.Lessons {
		if !l.Broken {
			v.lesson(l)
		}
	}
	if s.HasQuiz && !s.QuizBroken {
		v.quiz(s)
	}
}

func (v *validator) lesson(l LessonDoc) {
	p, m := l.Path, l.Meta
	v.requireID(p, "id", m.ID)
	if strings.TrimSpace(m.Title) == "" {
		v.errs.add(p, "frontmatter: title is required")
	}
	if strings.TrimSpace(m.Summary) == "" {
		v.errs.add(p, "frontmatter: summary is required")
	}
	if m.EstimatedMinutes <= 0 {
		v.errs.add(p, "frontmatter: estimated_minutes must be a positive number")
	}
	if len(m.Topics) == 0 {
		v.errs.add(p, "frontmatter: topics must list at least one tag")
	}
	for _, t := range m.Topics {
		if strings.TrimSpace(t) == "" {
			v.errs.add(p, "frontmatter: topics contains an empty tag")
		}
	}
	if strings.TrimSpace(l.Body) == "" {
		v.errs.add(p, "lesson body is empty")
	}
	seen := map[string]bool{}
	for _, pre := range m.Prerequisites {
		switch {
		case seen[pre]:
			v.errs.add(p, "frontmatter: prerequisite %q is listed more than once", pre)
		case pre == m.ID:
			v.errs.add(p, "frontmatter: lesson lists itself as a prerequisite")
		default:
			if _, ok := v.lessonPath[pre]; !ok && !v.anyBrokenLesson() {
				v.errs.add(p, "frontmatter: prerequisite %q does not exist", pre)
			}
		}
		seen[pre] = true
	}
}

func (v *validator) anyBrokenLesson() bool {
	for _, s := range v.b.Sections {
		for _, l := range s.Lessons {
			if l.Broken {
				return true
			}
		}
	}
	return false
}

// prerequisiteCycles reports each distinct cycle once, at the lesson file where it was found.
func (v *validator) prerequisiteCycles() {
	graph := map[string][]string{}
	var order []string
	for _, s := range v.b.Sections {
		for _, l := range s.Lessons {
			if l.Broken || l.Meta.ID == "" {
				continue
			}
			if _, dup := graph[l.Meta.ID]; dup {
				continue
			}
			graph[l.Meta.ID] = l.Meta.Prerequisites
			order = append(order, l.Meta.ID)
		}
	}

	const (
		unseen = iota
		inProgress
		done
	)
	state := map[string]int{}
	var stack []string
	reported := map[string]bool{}

	var visit func(id string)
	visit = func(id string) {
		state[id] = inProgress
		stack = append(stack, id)
		for _, dep := range graph[id] {
			if _, known := graph[dep]; !known {
				continue
			}
			switch state[dep] {
			case unseen:
				visit(dep)
			case inProgress:
				cycle := append([]string(nil), stack[slices.Index(stack, dep):]...)
				key := canonicalCycle(cycle)
				if !reported[key] {
					reported[key] = true
					v.errs.add(v.lessonPath[dep], "prerequisite cycle: %s -> %s", strings.Join(cycle, " -> "), dep)
				}
			}
		}
		stack = stack[:len(stack)-1]
		state[id] = done
	}
	for _, id := range order {
		if state[id] == unseen {
			visit(id)
		}
	}
}

// canonicalCycle rotates a cycle to start at its smallest id so each cycle has one key.
func canonicalCycle(c []string) string {
	min := 0
	for i := range c {
		if c[i] < c[min] {
			min = i
		}
	}
	rot := append(append([]string(nil), c[min:]...), c[:min]...)
	return strings.Join(rot, ">")
}

func (v *validator) requireID(file, field, id string) {
	switch {
	case id == "":
		v.errs.add(file, "%s is required", field)
	case !idPattern.MatchString(id):
		v.errs.add(file, "%s %q must be lowercase letters, digits and single hyphens (e.g. \"my-lesson\")", field, id)
	}
}

// questionIDs enforces uniqueness of question ids across all sections.
func (v *validator) questionIDs() {
	seen := map[string]string{}
	for _, s := range v.b.Sections {
		if !s.HasQuiz || s.QuizBroken {
			continue
		}
		for _, q := range s.Quiz {
			if q.ID == "" {
				continue
			}
			if prev, dup := seen[q.ID]; dup {
				where := "also defined in " + prev
				if prev == s.QuizPath {
					where = "defined twice in this file"
				}
				v.errs.add(s.QuizPath, "duplicate question id %q (%s)", q.ID, where)
			} else {
				seen[q.ID] = s.QuizPath
			}
		}
	}
}

func (v *validator) quiz(s SectionBundle) {
	p := s.QuizPath
	if len(s.Quiz) == 0 {
		v.errs.add(p, "quiz has no questions")
		return
	}
	for i, q := range s.Quiz {
		name := q.ID
		if name == "" {
			name = "#" + strconv.Itoa(i+1)
		}
		qe := func(format string, args ...any) {
			v.errs.add(p, "question "+name+": "+format, args...)
		}
		if q.ID == "" {
			qe("id is required")
		} else {
			if !idPattern.MatchString(q.ID) {
				qe("id must be lowercase letters, digits and single hyphens")
			}
		}
		v.question(q, qe)
	}
}

func (v *validator) question(q Question, qe func(string, ...any)) {
	if strings.TrimSpace(q.Topic) == "" {
		qe("topic is required")
	}
	if q.Difficulty < 1 || q.Difficulty > 3 {
		qe("difficulty must be 1, 2 or 3 (got %d)", q.Difficulty)
	}
	if strings.TrimSpace(q.Prompt) == "" {
		qe("prompt is required")
	}
	if strings.TrimSpace(q.Explanation) == "" {
		qe("explanation is required")
	}
	if q.LessonRef != "" {
		if _, ok := v.lessonPath[q.LessonRef]; !ok && !v.anyBrokenLesson() {
			qe("lesson_ref %q does not match any lesson id", q.LessonRef)
		}
	}

	// Options: ids unique and well-formed, text present.
	optIDs := map[string]bool{}
	for _, o := range q.Options {
		switch {
		case o.ID == "":
			qe("an option is missing its id")
			continue
		case !idPattern.MatchString(o.ID):
			qe("option id %q must be lowercase letters, digits and single hyphens", o.ID)
		}
		if optIDs[o.ID] {
			qe("duplicate option id %q", o.ID)
		}
		optIDs[o.ID] = true
		if strings.TrimSpace(o.Text) == "" {
			qe("option %q has no text", o.ID)
		}
	}

	// Correct answers must reference real options and not repeat.
	checkRefs := func() bool {
		ok := true
		seen := map[string]bool{}
		for _, c := range q.Correct {
			if !optIDs[c] {
				qe("correct id %q is not one of the option ids", c)
				ok = false
			}
			if seen[c] {
				qe("correct id %q is listed more than once", c)
				ok = false
			}
			seen[c] = true
		}
		return ok
	}

	switch q.Type {
	case SingleChoice:
		v.minOptions(q, qe)
		if len(q.Correct) != 1 {
			qe("single_choice needs exactly one correct option (got %d)", len(q.Correct))
		}
		checkRefs()
	case MultiChoice:
		v.minOptions(q, qe)
		if len(q.Correct) < 2 {
			qe("multi_choice needs at least two correct options (got %d); use single_choice for one", len(q.Correct))
		}
		if len(q.Options) > 0 && len(q.Correct) >= len(q.Options) {
			qe("multi_choice cannot have every option correct")
		}
		checkRefs()
	case TrueFalse:
		if len(q.Options) > 0 && !(len(q.Options) == 2 && optIDs["true"] && optIDs["false"]) {
			qe("true_false options must be exactly ids \"true\" and \"false\" (or omitted)")
		}
		if len(q.Correct) != 1 || (q.Correct[0] != "true" && q.Correct[0] != "false") {
			qe("true_false needs correct: true or correct: false")
		}
	case Ordering:
		v.minOptions(q, qe)
		if checkRefs() && len(q.Correct) != len(q.Options) {
			qe("ordering correct must list every option exactly once (%d listed, %d options)", len(q.Correct), len(q.Options))
		}
	case "":
		qe("type is required (single_choice, multi_choice, true_false or ordering)")
	default:
		qe("unknown type %q (want single_choice, multi_choice, true_false or ordering)", q.Type)
	}
}

func (v *validator) minOptions(q Question, qe func(string, ...any)) {
	if len(q.Options) < 2 {
		qe("%s needs at least two options (got %d)", q.Type, len(q.Options))
	}
}

// componentRefs checks that <Diagram id="..."/> and <Term id="..."/> in lessons point at real definitions.
func (v *validator) componentRefs(anyBrokenDiagram, glossaryBroken bool) {
	for _, s := range v.b.Sections {
		for _, l := range s.Lessons {
			if l.Broken {
				continue
			}
			at := func(line int) string { return fmt.Sprintf("line %d", line+l.BodyLine) }
			for _, r := range scanTags(l.Body, "Diagram") {
				switch {
				case !r.HasID || r.ID == "":
					v.errs.add(l.Path, "%s: <Diagram> needs an id=\"...\" attribute", at(r.Line))
				case v.diagramPath[r.ID] == "" && !anyBrokenDiagram:
					v.errs.add(l.Path, "%s: <Diagram id=%q> does not match any diagram in diagrams/", at(r.Line), r.ID)
				}
			}
			for _, r := range scanTags(l.Body, "Term") {
				switch {
				case !r.HasID || r.ID == "":
					v.errs.add(l.Path, "%s: <Term> needs an id=\"...\" attribute", at(r.Line))
				case !v.termIDs[r.ID] && !glossaryBroken:
					v.errs.add(l.Path, "%s: <Term id=%q> does not match any term in glossary.yaml", at(r.Line), r.ID)
				}
			}
		}
	}
}
