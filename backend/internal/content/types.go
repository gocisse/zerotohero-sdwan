// Package content loads, validates and serves the course content in /content.
package content

import (
	"fmt"

	"go.yaml.in/yaml/v3"
)

// QuestionType enumerates supported quiz question types.
type QuestionType string

const (
	SingleChoice QuestionType = "single_choice"
	MultiChoice  QuestionType = "multi_choice"
	TrueFalse    QuestionType = "true_false"
	Ordering     QuestionType = "ordering"
)

// StringList decodes either a YAML scalar or a sequence into a slice.
type StringList []string

func (s *StringList) UnmarshalYAML(n *yaml.Node) error {
	switch n.Kind {
	case yaml.ScalarNode:
		*s = StringList{n.Value}
	case yaml.SequenceNode:
		out := make(StringList, 0, len(n.Content))
		for _, c := range n.Content {
			if c.Kind != yaml.ScalarNode {
				return fmt.Errorf("line %d: expected a list of ids", c.Line)
			}
			out = append(out, c.Value)
		}
		*s = out
	default:
		return fmt.Errorf("line %d: expected an id or a list of ids", n.Line)
	}
	return nil
}

// CourseMeta is the content of course.yaml.
type CourseMeta struct {
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	Sections    []string `yaml:"sections"`
}

// SectionMeta is the content of a section's section.yaml.
type SectionMeta struct {
	ID               string   `yaml:"id"`
	Title            string   `yaml:"title"`
	Summary          string   `yaml:"summary"`
	EstimatedMinutes int      `yaml:"estimated_minutes"`
	Placeholder      bool     `yaml:"placeholder"`
	Lessons          []string `yaml:"lessons"`
}

// LessonMeta is a lesson's YAML frontmatter.
type LessonMeta struct {
	ID               string   `yaml:"id"`
	Title            string   `yaml:"title"`
	Summary          string   `yaml:"summary"`
	EstimatedMinutes int      `yaml:"estimated_minutes"`
	Topics           []string `yaml:"topics"`
	Prerequisites    []string `yaml:"prerequisites"`
}

// Option is one answer choice. Explanation is optional and shown for wrong picks.
type Option struct {
	ID          string `yaml:"id"`
	Text        string `yaml:"text"`
	Explanation string `yaml:"explanation"`
}

// Question is one quiz question. For ordering questions, Correct is the ordered ids.
type Question struct {
	ID          string       `yaml:"id"`
	Topic       string       `yaml:"topic"`
	Difficulty  int          `yaml:"difficulty"`
	Type        QuestionType `yaml:"type"`
	Prompt      string       `yaml:"prompt"`
	Options     []Option     `yaml:"options"`
	Correct     StringList   `yaml:"correct"`
	Explanation string       `yaml:"explanation"`
	LessonRef   string       `yaml:"lesson_ref"`
}

// Course is the validated, in-memory course.
type Course struct {
	Meta     CourseMeta
	Sections []*Section // ordered as in course.yaml
	Glossary []GlossaryTerm
	lessons  map[string]*Lesson
	sections map[string]*Section
	diagrams map[string]*Diagram
}

// Section is a validated section with its lessons in order.
type Section struct {
	Meta    SectionMeta
	Dir     string
	Lessons []*Lesson
	Quiz    []Question
}

// Lesson is a validated lesson. Body is the raw MDX without frontmatter.
type Lesson struct {
	Meta      LessonMeta
	SectionID string
	Path      string
	Body      string
}

// Lesson returns the lesson with the given id.
func (c *Course) Lesson(id string) (*Lesson, bool) { l, ok := c.lessons[id]; return l, ok }

// Section returns the section with the given id.
func (c *Course) Section(id string) (*Section, bool) { s, ok := c.sections[id]; return s, ok }

// Diagram returns the diagram with the given id (packet routes resolved).
func (c *Course) Diagram(id string) (*Diagram, bool) { d, ok := c.diagrams[id]; return d, ok }

// DiagramCount is the number of diagrams in the course.
func (c *Course) DiagramCount() int { return len(c.diagrams) }
