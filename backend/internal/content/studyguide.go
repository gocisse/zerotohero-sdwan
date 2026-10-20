package content

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"strings"
)

// StudyGuide is content/study-guide.yaml: the learner's study guide (navigation labels only) and what the course offers
// for each of its topics.
type StudyGuide struct {
	Beyond          GuideBeyond    `yaml:"beyond" json:"beyond"`
	UnmappedLessons []string       `yaml:"unmapped_lessons" json:"unmapped_lessons"`
	Sections        []GuideSection `yaml:"sections" json:"sections"`
}

// GuideBeyond is the group for glossary terms that belong to no guide topic, with the reason shown to the learner.
type GuideBeyond struct {
	Title  string `yaml:"title" json:"title"`
	Reason string `yaml:"reason" json:"reason"`
}

// GuideSection is one section (G1..G5) of the study guide.
type GuideSection struct {
	ID     string       `yaml:"id" json:"id"`
	Title  string       `yaml:"title" json:"title"`
	Topics []GuideTopic `yaml:"topics" json:"topics"`
}

// GuideTopic is one topic of the study guide and what the course offers for it.
type GuideTopic struct {
	ID         string   `yaml:"id" json:"id"`
	Title      string   `yaml:"title" json:"title"`
	Kind       string   `yaml:"kind" json:"kind"`
	Coverage   string   `yaml:"coverage" json:"coverage"`
	Note       string   `yaml:"note" json:"note"`
	Lessons    []string `yaml:"lessons" json:"lessons"`
	QuizTopics []string `yaml:"quiz_topics" json:"quiz_topics"`
	Diagrams   []string `yaml:"diagrams" json:"diagrams"`
	Labs       []string `yaml:"labs" json:"labs"`
	SimScreens []string `yaml:"sim_screens" json:"sim_screens"`
}

// Guide coverage levels and topic kinds.
const (
	CoverageFull    = "full"
	CoveragePartial = "partial"
	CoverageNone    = "none"
)

// SimScreen is a simulator screen a guide topic can point to. Path holds "{topology}" for screens inside a topology.
type SimScreen struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Path  string `json:"path"`
}

// SimScreens are the simulator routes (see App.tsx) a study-guide topic may reference.
var SimScreens = []SimScreen{
	{"edges", "Edge list and Edge tabs", "/sim/{topology}"},
	{"what-if", "What-if", "/sim/{topology}/what-if"},
	{"challenge", "Configuration challenge", "/sim/{topology}/challenge"},
	{"map", "Tunnel map", "/sim/{topology}/map"},
	{"monitor", "Monitor (Links, Paths)", "/sim/{topology}/monitor"},
	{"paths", "Path explainer", "/sim/{topology}/paths"},
	{"timeline", "Timeline", "/sim/{topology}/timeline"},
	{"overlay-challenge", "Overlay challenge", "/sim/{topology}/overlay-challenge"},
	{"routes", "Routes", "/sim/{topology}/routes"},
	{"reachability", "Reachability explainer", "/sim/{topology}/reachability"},
	{"routing-challenge", "Routing challenge", "/sim/{topology}/routing-challenge"},
	{"trace", "Flow tracer", "/sim/{topology}/trace"},
	{"labs", "Labs", "/sim/labs"},
	{"self-check", "Self-check", "/sim/self-check"},
}

var guideIDPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*(-[a-z0-9]+)*$`)

// readGuide reads content/study-guide.yaml, which is optional.
func readGuide(fsys fs.FS, b *Bundle, errs *ErrorList) {
	b.GuidePath = "study-guide.yaml"
	data, err := fs.ReadFile(fsys, b.GuidePath)
	switch {
	case err == nil:
		b.HasGuide = true
		if err := decodeStrict(data, &b.Guide); err != nil {
			errs.add(b.GuidePath, "%v", err)
			b.GuideBroken = true
		}
	case !errors.Is(err, fs.ErrNotExist):
		errs.add(b.GuidePath, "cannot read file: %v", unwrapPathErr(err))
		b.GuideBroken = true
	}
}

// checkGuide validates the study guide against the rest of the bundle. Errors fail the content; warnings are shown.
func checkGuide(b *Bundle) (errs, warns ErrorList) {
	p := b.GuidePath
	termsUseGuide := false
	if b.HasGlossary && !b.GlossaryBad {
		for _, t := range b.Glossary {
			if len(t.GuideTopics) > 0 || t.BeyondGuide {
				termsUseGuide = true
			}
		}
	}
	if !b.HasGuide {
		if termsUseGuide {
			errs.add(b.GlossaryPath, "glossary terms use guide_topics or beyond_guide but there is no study-guide.yaml")
		}
		return
	}
	if b.GuideBroken {
		return
	}
	g := b.Guide

	// What the guide may reference. A file that failed to parse could be the cause of an "unknown" error: skip those checks.
	lessons, quizTopics, diagrams, labs := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	var lessonOrder []string
	lessonsBroken, quizBroken, diagramsBroken, labsBroken := false, false, false, false
	for _, s := range b.Sections {
		for _, l := range s.Lessons {
			if l.Broken {
				lessonsBroken = true
			} else {
				lessons[l.Meta.ID] = true
				lessonOrder = append(lessonOrder, l.Meta.ID)
			}
		}
		if s.QuizBroken {
			quizBroken = true
		}
		for _, q := range s.Quiz {
			quizTopics[q.Topic] = true
		}
	}
	for _, d := range b.Diagrams {
		if d.Broken {
			diagramsBroken = true
		} else {
			diagrams[d.D.ID] = true
		}
	}
	for _, d := range b.Labs {
		if d.Broken || d.Lab == nil {
			labsBroken = true
		} else {
			labs[d.Lab.ID] = true
		}
	}
	screens := map[string]bool{}
	for _, s := range SimScreens {
		screens[s.ID] = true
	}

	if strings.TrimSpace(g.Beyond.Title) == "" {
		errs.add(p, "beyond.title is required")
	}
	if strings.TrimSpace(g.Beyond.Reason) == "" {
		errs.add(p, "beyond.reason is required (say why those terms have no guide topic)")
	}
	if len(g.Sections) == 0 {
		errs.add(p, "sections must list at least one section")
	}

	sectionIDs, topicIDs := map[string]bool{}, map[string]string{}
	referenced := map[string]bool{}
	for si, s := range g.Sections {
		name := fmt.Sprintf("section %q", s.ID)
		switch {
		case s.ID == "":
			name = fmt.Sprintf("sections[%d]", si)
			errs.add(p, "%s: id is required", name)
		case !guideIDPattern.MatchString(s.ID):
			errs.add(p, "%s: id must be letters and digits (for example G1)", name)
		case sectionIDs[s.ID]:
			errs.add(p, "%s: duplicate section id", name)
		}
		sectionIDs[s.ID] = true
		if strings.TrimSpace(s.Title) == "" {
			errs.add(p, "%s: title is required", name)
		}
		if len(s.Topics) == 0 {
			errs.add(p, "%s: topics must list at least one topic", name)
		}
		for ti, t := range s.Topics {
			tn := fmt.Sprintf("topic %q", t.ID)
			if t.ID == "" {
				tn = fmt.Sprintf("%s topics[%d]", name, ti)
				errs.add(p, "%s: id is required", tn)
			} else {
				if !idPattern.MatchString(t.ID) {
					errs.add(p, "%s: id must be lowercase letters, digits and single hyphens", tn)
				}
				if prev, dup := topicIDs[t.ID]; dup {
					errs.add(p, "%s: duplicate topic id (also in %s)", tn, prev)
				}
				topicIDs[t.ID] = name
			}
			if strings.TrimSpace(t.Title) == "" {
				errs.add(p, "%s: title is required", tn)
			}
			switch t.Kind {
			case "concept", "config", "topology":
			default:
				errs.add(p, "%s: kind must be concept, config or topology (got %q)", tn, t.Kind)
			}
			if strings.TrimSpace(t.Note) == "" {
				errs.add(p, "%s: note is required (one sentence saying what is and is not covered)", tn)
			}
			switch t.Coverage {
			case CoverageFull:
				if len(t.Lessons) == 0 {
					errs.add(p, "%s: coverage full needs at least one lesson", tn)
				}
			case CoveragePartial:
				if len(t.Lessons) == 0 {
					errs.add(p, "%s: coverage partial needs at least one lesson (without one, use none)", tn)
				}
			case CoverageNone:
				if len(t.Lessons) > 0 {
					errs.add(p, "%s: coverage none must have no lessons (it lists %s)", tn, strings.Join(t.Lessons, ", "))
				}
			default:
				errs.add(p, "%s: coverage must be full, partial or none (got %q)", tn, t.Coverage)
			}
			refs := []struct {
				field  string
				ids    []string
				known  map[string]bool
				broken bool
			}{
				{"lessons", t.Lessons, lessons, lessonsBroken},
				{"quiz_topics", t.QuizTopics, quizTopics, quizBroken},
				{"diagrams", t.Diagrams, diagrams, diagramsBroken},
				{"labs", t.Labs, labs, labsBroken},
				{"sim_screens", t.SimScreens, screens, false},
			}
			for _, r := range refs {
				seen := map[string]bool{}
				for _, id := range r.ids {
					switch {
					case seen[id]:
						errs.add(p, "%s: %s lists %q more than once", tn, r.field, id)
					case !r.known[id] && !r.broken:
						errs.add(p, "%s: %s %q does not exist", tn, r.field, id)
					case r.field == "labs" && strings.HasPrefix(id, "self-check-"):
						errs.add(p, "%s: labs %q is not a lab (self-check screens are not listed)", tn, id)
					}
					seen[id] = true
				}
			}
			for _, id := range t.Lessons {
				referenced[id] = true
			}
		}
	}

	listed := map[string]bool{}
	for _, id := range g.UnmappedLessons {
		switch {
		case listed[id]:
			errs.add(p, "unmapped_lessons lists %q more than once", id)
		case !lessons[id] && !lessonsBroken:
			errs.add(p, "unmapped_lessons %q does not exist", id)
		case referenced[id]:
			errs.add(p, "unmapped_lessons %q is referenced by a topic, so it is mapped", id)
		}
		listed[id] = true
	}
	if !lessonsBroken {
		for _, id := range lessonOrder {
			if !referenced[id] && !listed[id] {
				warns.add(p, "lesson %q is not referenced by any topic and not listed in unmapped_lessons", id)
			}
		}
	}

	if b.HasGlossary && !b.GlossaryBad {
		for _, t := range b.Glossary {
			switch {
			case t.BeyondGuide && len(t.GuideTopics) > 0:
				errs.add(b.GlossaryPath, "term %s: has both guide_topics and beyond_guide (choose one)", t.ID)
			case !t.BeyondGuide && len(t.GuideTopics) == 0:
				errs.add(b.GlossaryPath, "term %s: needs guide_topics or beyond_guide: true", t.ID)
			}
			seen := map[string]bool{}
			for _, id := range t.GuideTopics {
				switch {
				case seen[id]:
					errs.add(b.GlossaryPath, "term %s: guide_topics lists %q more than once", t.ID, id)
				case topicIDs[id] == "":
					errs.add(b.GlossaryPath, "term %s: guide_topics %q is not a topic in study-guide.yaml", t.ID, id)
				}
				seen[id] = true
			}
		}
	}
	return
}

// GuideWarnings are the study guide's non-fatal findings (for example lessons no topic maps).
func (c *Course) GuideWarnings() ErrorList { return c.guideWarnings }

// StudyGuide returns the study guide, or nil if the content has none.
func (c *Course) StudyGuide() *StudyGuide { return c.guide }
