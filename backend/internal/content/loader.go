package content

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// allowedRootDirs are the top-level directories permitted next to course.yaml.
// "labs" is reserved for the future simulator scenarios; "topologies" holds the simulator's teaching topologies.
var allowedRootDirs = map[string]bool{"sections": true, "labs": true, "diagrams": true, "reference": true, "topologies": true}

// Load reads, validates and builds the course from fsys (rooted at the content dir).
// On failure the returned error is an ErrorList containing every problem found.
func Load(fsys fs.FS) (*Course, error) {
	b, errs := ReadBundle(fsys)
	errs = append(errs, Validate(b)...)
	if len(errs) > 0 {
		return nil, errs
	}
	return Build(b), nil
}

// ReadBundle parses every content file. It reports syntax-level problems only;
// call Validate for cross-file rules.
func ReadBundle(fsys fs.FS) (*Bundle, ErrorList) {
	var errs ErrorList
	b := &Bundle{CoursePath: "course.yaml"}

	if data, err := fs.ReadFile(fsys, b.CoursePath); err != nil {
		errs.add(b.CoursePath, "cannot read file: %v", unwrapPathErr(err))
		b.CourseBroken = true
	} else if err := decodeStrict(data, &b.Course); err != nil {
		errs.add(b.CoursePath, "%v", err)
		b.CourseBroken = true
	}

	rootEntries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		errs.add(".", "cannot read content directory: %v", err)
		return b, errs
	}
	for _, e := range rootEntries {
		name := e.Name()
		switch {
		case ignored(name):
		case e.IsDir() && allowedRootDirs[name]:
		case e.IsDir():
			b.Extra = append(b.Extra, name+"/")
		case name != "course.yaml" && name != "README.md" && name != "glossary.yaml":
			b.Extra = append(b.Extra, name)
		}
	}

	b.GlossaryPath = "glossary.yaml"
	if data, err := fs.ReadFile(fsys, b.GlossaryPath); err == nil {
		b.HasGlossary = true
		if err := decodeStrict(data, &b.Glossary); err != nil {
			errs.add(b.GlossaryPath, "%v", err)
			b.GlossaryBad = true
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		errs.add(b.GlossaryPath, "cannot read file: %v", unwrapPathErr(err))
		b.GlossaryBad = true
	}

	diagramEntries, _ := fs.ReadDir(fsys, "diagrams")
	for _, e := range diagramEntries {
		p := path.Join("diagrams", e.Name())
		switch {
		case ignored(e.Name()):
		case !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml"):
			doc := DiagramDoc{Path: p}
			if data, err := fs.ReadFile(fsys, p); err != nil {
				errs.add(p, "cannot read file: %v", unwrapPathErr(err))
				doc.Broken = true
			} else if err := decodeStrict(data, &doc.D); err != nil {
				errs.add(p, "%v", err)
				doc.Broken = true
			}
			b.Diagrams = append(b.Diagrams, doc)
		default:
			b.Extra = append(b.Extra, p)
		}
	}

	readTopologies(fsys, b, &errs)
	readLabs(fsys, b, &errs)

	b.References = readReferences(fsys, b, &errs)

	sectionEntries, err := fs.ReadDir(fsys, "sections")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		errs.add("sections", "cannot read directory: %v", err)
	}
	for _, e := range sectionEntries {
		if ignored(e.Name()) {
			continue
		}
		dir := path.Join("sections", e.Name())
		if !e.IsDir() {
			b.Extra = append(b.Extra, dir)
			continue
		}
		b.Sections = append(b.Sections, readSection(fsys, dir, b, &errs))
	}
	sort.Strings(b.Extra)
	return b, errs
}

func readSection(fsys fs.FS, dir string, b *Bundle, errs *ErrorList) SectionBundle {
	sb := SectionBundle{
		Dir:      dir,
		MetaPath: path.Join(dir, "section.yaml"),
		QuizPath: path.Join(dir, "quiz.yaml"),
	}

	if data, err := fs.ReadFile(fsys, sb.MetaPath); err != nil {
		errs.add(sb.MetaPath, "cannot read file: %v", unwrapPathErr(err))
		sb.MetaBroken = true
	} else if err := decodeStrict(data, &sb.Meta); err != nil {
		errs.add(sb.MetaPath, "%v", err)
		sb.MetaBroken = true
	}

	if data, err := fs.ReadFile(fsys, sb.QuizPath); err == nil {
		sb.HasQuiz = true
		if err := decodeStrict(data, &sb.Quiz); err != nil {
			errs.add(sb.QuizPath, "%v", err)
			sb.QuizBroken = true
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		errs.add(sb.QuizPath, "cannot read file: %v", unwrapPathErr(err))
		sb.QuizBroken = true
	}

	entries, _ := fs.ReadDir(fsys, dir)
	for _, e := range entries {
		name := e.Name()
		p := path.Join(dir, name)
		switch {
		case ignored(name), name == "section.yaml", name == "quiz.yaml":
		case e.IsDir() && name == "lessons":
			lessonEntries, _ := fs.ReadDir(fsys, p)
			for _, le := range lessonEntries {
				lp := path.Join(p, le.Name())
				switch {
				case ignored(le.Name()):
				case !le.IsDir() && strings.HasSuffix(le.Name(), ".mdx"):
					sb.Lessons = append(sb.Lessons, readLesson(fsys, lp, errs))
				default:
					b.Extra = append(b.Extra, lp)
				}
			}
		default:
			suffix := ""
			if e.IsDir() {
				suffix = "/"
			}
			b.Extra = append(b.Extra, p+suffix)
		}
	}
	return sb
}

func readLesson(fsys fs.FS, p string, errs *ErrorList) LessonDoc {
	doc := LessonDoc{Path: p}
	data, err := fs.ReadFile(fsys, p)
	if err != nil {
		errs.add(p, "cannot read file: %v", unwrapPathErr(err))
		doc.Broken = true
		return doc
	}
	front, body, offset, err := splitFrontmatter(data)
	if err == nil {
		err = decodeStrict(front, &doc.Meta)
	}
	if err != nil {
		errs.add(p, "frontmatter: %v", err)
		doc.Broken = true
		return doc
	}
	doc.Body, doc.BodyLine = body, offset
	return doc
}

// splitFrontmatter separates a leading "---" delimited YAML block from the MDX body.
func splitFrontmatter(data []byte) (front []byte, body string, bodyLine int, err error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.TrimPrefix(text, "\xef\xbb\xbf")
	rest, ok := strings.CutPrefix(text, "---\n")
	if !ok {
		return nil, "", 0, errors.New("file must start with a '---' YAML frontmatter block")
	}
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return nil, "", 0, errors.New("frontmatter is not closed with a '---' line")
	}
	after := rest[end+len("\n---"):]
	if nl := strings.IndexByte(after, '\n'); nl >= 0 {
		if strings.TrimSpace(after[:nl]) != "" {
			return nil, "", 0, errors.New("frontmatter closing '---' must be on its own line")
		}
		after = after[nl+1:]
	} else if strings.TrimSpace(after) != "" {
		return nil, "", 0, errors.New("frontmatter closing '---' must be on its own line")
	} else {
		after = ""
	}
	return []byte(rest[:end]), after, strings.Count(text, "\n") - strings.Count(after, "\n"), nil
}

// decodeStrict decodes YAML, rejecting unknown fields so typos are caught.
func decodeStrict(data []byte, v any) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(v); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("file is empty")
		}
		return err
	}
	return nil
}

func ignored(name string) bool { return strings.HasPrefix(name, ".") }

func unwrapPathErr(err error) error {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err
	}
	return err
}

// Build assembles a Course from a bundle that has already passed Validate.
func Build(b *Bundle) *Course {
	c := &Course{
		Meta:       b.Course,
		Glossary:   append([]GlossaryTerm{}, b.Glossary...),
		lessons:    map[string]*Lesson{},
		sections:   map[string]*Section{},
		diagrams:   map[string]*Diagram{},
		topologies: buildTopologies(b.Topologies),
		labs:       buildLabs(b.Labs),
	}
	for _, d := range b.Diagrams {
		c.diagrams[d.D.ID] = d.D.normalized()
	}
	for i := range c.Glossary {
		c.Glossary[i].Aliases = append([]string{}, c.Glossary[i].Aliases...)
		c.Glossary[i].Related = append([]string{}, c.Glossary[i].Related...)
	}
	byID := map[string]*SectionBundle{}
	for i := range b.Sections {
		byID[b.Sections[i].Meta.ID] = &b.Sections[i]
	}
	for _, sid := range b.Course.Sections {
		sb := byID[sid]
		s := &Section{Meta: sb.Meta, Dir: sb.Dir, Quiz: sb.Quiz}
		docs := map[string]*LessonDoc{}
		for i := range sb.Lessons {
			docs[sb.Lessons[i].Meta.ID] = &sb.Lessons[i]
		}
		for _, lid := range sb.Meta.Lessons {
			d := docs[lid]
			l := &Lesson{Meta: d.Meta, SectionID: sid, Path: d.Path, Body: d.Body}
			s.Lessons = append(s.Lessons, l)
			c.lessons[lid] = l
		}
		for i := range s.Quiz {
			fillTrueFalse(&s.Quiz[i])
		}
		c.Sections = append(c.Sections, s)
		c.sections[sid] = s
	}
	return c
}

// fillTrueFalse supplies the standard options for a true_false question that omits them.
func fillTrueFalse(q *Question) {
	if q.Type == TrueFalse && len(q.Options) == 0 {
		q.Options = []Option{{ID: "true", Text: "True"}, {ID: "false", Text: "False"}}
	}
}

// readReferences reads the fact sheets and observation files. Reference fact sheets are author material: not
// orphans, but every file must carry a claims table.
func readReferences(fsys fs.FS, b *Bundle, errs *ErrorList) []ReferenceDoc {
	var refs []ReferenceDoc
	refEntries, _ := fs.ReadDir(fsys, "reference")
	for _, e := range refEntries {
		p := path.Join("reference", e.Name())
		if ignored(e.Name()) || e.Name() == "README.md" {
			continue
		}
		doc := ReferenceDoc{Path: p}
		if e.IsDir() && e.Name() == "observations" {
			// User observations from a real or lab VCO: one claims table per note file.
			obs, _ := fs.ReadDir(fsys, p)
			for _, o := range obs {
				op := path.Join(p, o.Name())
				if ignored(o.Name()) || o.Name() == "README.md" {
					continue
				}
				if o.IsDir() {
					b.Extra = append(b.Extra, op+"/")
					continue
				}
				od := ReferenceDoc{Path: op}
				if data, err := fs.ReadFile(fsys, op); err != nil {
					errs.add(op, "cannot read file: %v", unwrapPathErr(err))
				} else {
					od.Content = string(data)
				}
				refs = append(refs, od)
			}
			continue
		}
		if e.IsDir() {
			b.Extra = append(b.Extra, p+"/")
			continue
		}
		if data, err := fs.ReadFile(fsys, p); err != nil {
			errs.add(p, "cannot read file: %v", unwrapPathErr(err))
		} else {
			doc.Content = string(data)
		}
		refs = append(refs, doc)
	}
	return refs
}
