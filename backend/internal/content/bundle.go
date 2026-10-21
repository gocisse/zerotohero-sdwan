package content

// Bundle is the raw, parsed content of the content directory before cross-file
// validation. Files that failed to parse are flagged Broken so the validator
// skips checks that would only produce misleading follow-on errors.
type Bundle struct {
	CoursePath   string
	Course       CourseMeta
	CourseBroken bool
	Sections     []SectionBundle
	Diagrams     []DiagramDoc
	References   []ReferenceDoc
	Topologies   []TopologyDoc
	Labs         []LabDoc
	Runbooks     []RunbookDoc
	GlossaryPath string
	HasGlossary  bool
	Glossary     []GlossaryTerm
	GlossaryBad  bool
	GuidePath    string
	HasGuide     bool
	Guide        StudyGuide
	GuideBroken  bool
	Extra        []string // unexpected files, relative to the content dir
}

// SectionBundle is one directory under sections/.
type SectionBundle struct {
	Dir        string // e.g. "sections/00-sample"
	MetaPath   string
	Meta       SectionMeta
	MetaBroken bool
	Lessons    []LessonDoc
	QuizPath   string
	HasQuiz    bool
	Quiz       []Question
	QuizBroken bool
}

// ReferenceDoc is one file in content/reference/ (author fact sheets, never shown to learners).
type ReferenceDoc struct {
	Path    string
	Content string
}

// DiagramDoc is one diagrams/*.yaml file.
type DiagramDoc struct {
	Path   string
	D      Diagram
	Broken bool
}

// LessonDoc is one lessons/*.mdx file.
type LessonDoc struct {
	Path     string
	Meta     LessonMeta
	Body     string
	BodyLine int // lines of frontmatter before the body, for reporting line numbers in the file
	Broken   bool
}
