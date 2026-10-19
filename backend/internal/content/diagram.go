package content

import (
	"fmt"
	"slices"
	"strings"
)

// NodeKinds are the icon kinds the frontend can draw.
var NodeKinds = []string{"vce", "vco", "vcg", "router", "l3switch", "l2switch", "firewall", "host", "cloud", "internet", "mpls", "datacenter", "saas", "generic"}

// LinkStyles are the link looks defined once in the frontend style map (and shown in the legend).
var LinkStyles = []string{"underlay-internet", "underlay-mpls", "underlay-lte", "overlay-tunnel", "ipsec", "control", "lan"}

// Diagram is one content/diagrams/<id>.yaml file. Coordinates are in viewBox units.
type Diagram struct {
	ID     string         `yaml:"id" json:"id"`
	Title  string         `yaml:"title" json:"title"`
	Alt    string         `yaml:"alt" json:"alt"`
	Width  float64        `yaml:"width" json:"width"`
	Height float64        `yaml:"height" json:"height"`
	Nodes  []DiagramNode  `yaml:"nodes" json:"nodes"`
	Groups []DiagramGroup `yaml:"groups" json:"groups"`
	Links  []DiagramLink  `yaml:"links" json:"links"`
	Steps  []DiagramStep  `yaml:"steps" json:"steps"`
}

type DiagramNode struct {
	ID       string  `yaml:"id" json:"id"`
	Label    string  `yaml:"label" json:"label"`
	Kind     string  `yaml:"kind" json:"kind"`
	X        float64 `yaml:"x" json:"x"`
	Y        float64 `yaml:"y" json:"y"`
	Sublabel string  `yaml:"sublabel" json:"sublabel,omitempty"`
	Group    string  `yaml:"group" json:"group,omitempty"`
}

// DiagramGroup draws a site/region boundary: an explicit Box, or the padded bounds of its member nodes
// (listed in Nodes and/or pointing at the group through node.group).
type DiagramGroup struct {
	ID    string      `yaml:"id" json:"id"`
	Label string      `yaml:"label" json:"label"`
	Nodes []string    `yaml:"nodes" json:"nodes"`
	Box   *DiagramBox `yaml:"box" json:"box,omitempty"`
}

type DiagramBox struct {
	X      float64 `yaml:"x" json:"x"`
	Y      float64 `yaml:"y" json:"y"`
	Width  float64 `yaml:"width" json:"width"`
	Height float64 `yaml:"height" json:"height"`
}

type DiagramLink struct {
	ID    string  `yaml:"id" json:"id"`
	From  string  `yaml:"from" json:"from"`
	To    string  `yaml:"to" json:"to"`
	Style string  `yaml:"style" json:"style"`
	Label string  `yaml:"label" json:"label,omitempty"`
	Curve float64 `yaml:"curve" json:"curve,omitempty"` // perpendicular bend in viewBox units; sign picks the side
}

// DiagramStep is one stage of the walkthrough. Elements are node and link ids.
// An element listed in any step's Show starts hidden and appears at that step; Hide removes it again.
type DiagramStep struct {
	Title     string           `yaml:"title" json:"title"`
	Narration string           `yaml:"narration" json:"narration"`
	Highlight []string         `yaml:"highlight" json:"highlight"` // emphasized; when non-empty everything else dims
	Show      []string         `yaml:"show" json:"show"`
	Hide      []string         `yaml:"hide" json:"hide"`
	Packets   []DiagramPacket  `yaml:"packets" json:"packets"`
	Callouts  []DiagramCallout `yaml:"callouts" json:"callouts"`
}

// DiagramPacket animates along Path: node ids, or link ids between nodes (starting with a node).
type DiagramPacket struct {
	Path       []string `yaml:"path" json:"path"`
	Label      string   `yaml:"label" json:"label,omitempty"`
	ColorStyle string   `yaml:"color_style" json:"color_style,omitempty"`
	DelayMS    int      `yaml:"delay_ms" json:"delay_ms"`
	// Route is derived (not authored): the node and link sequence the path resolves to.
	Route *DiagramRoute `yaml:"-" json:"route,omitempty"`
}

type DiagramRoute struct {
	Nodes []string `json:"nodes"`
	Links []string `json:"links"`
}

type DiagramCallout struct {
	Target string `yaml:"target" json:"target"`
	Text   string `yaml:"text" json:"text"`
}

// resolveRoute turns an authored packet path into the node/link sequence it travels.
// A path alternates freely between node ids and link ids; a link id must touch the current node,
// and consecutive node ids must be directly joined by a link (either direction).
func (d *Diagram) resolveRoute(path []string) (*DiagramRoute, error) {
	nodes := map[string]bool{}
	for _, n := range d.Nodes {
		nodes[n.ID] = true
	}
	links := map[string]DiagramLink{}
	for _, l := range d.Links {
		links[l.ID] = l
	}
	if len(path) == 0 {
		return nil, fmt.Errorf("path is empty")
	}
	if !nodes[path[0]] {
		if _, isLink := links[path[0]]; isLink {
			return nil, fmt.Errorf("path must start with a node id, not link %q", path[0])
		}
		return nil, fmt.Errorf("unknown node or link id %q in path", path[0])
	}

	route := &DiagramRoute{Nodes: []string{path[0]}}
	cur := path[0]
	for _, el := range path[1:] {
		if l, isLink := links[el]; isLink {
			switch cur {
			case l.From:
				cur = l.To
			case l.To:
				cur = l.From
			default:
				return nil, fmt.Errorf("link %q does not touch %q", el, cur)
			}
			route.Links = append(route.Links, el)
			route.Nodes = append(route.Nodes, cur)
			continue
		}
		if !nodes[el] {
			return nil, fmt.Errorf("unknown node or link id %q in path", el)
		}
		if el == cur {
			continue // node id repeated after a link already landed there
		}
		var via string
		for _, l := range d.Links {
			if (l.From == cur && l.To == el) || (l.To == cur && l.From == el) {
				via = l.ID
				break
			}
		}
		if via == "" {
			return nil, fmt.Errorf("no link connects %q and %q (the path skips a link)", cur, el)
		}
		route.Links = append(route.Links, via)
		route.Nodes = append(route.Nodes, el)
		cur = el
	}
	if len(route.Links) == 0 {
		return nil, fmt.Errorf("path must travel along at least one link")
	}
	return route, nil
}

// validateDiagram checks one diagram file; problems are added to errs under path.
func validateDiagram(errs *ErrorList, path string, d Diagram) {
	add := func(format string, args ...any) { errs.add(path, format, args...) }

	if !idPattern.MatchString(d.ID) {
		add("id %q must be lowercase letters, digits and single hyphens (e.g. \"my-diagram\")", d.ID)
	}
	if strings.TrimSpace(d.Title) == "" {
		add("title is required")
	}
	if strings.TrimSpace(d.Alt) == "" {
		add("alt is required (a plain-text description of the diagram for screen readers)")
	}
	if d.Width <= 0 || d.Height <= 0 {
		add("width and height must be positive (got %vx%v)", d.Width, d.Height)
	}
	if len(d.Nodes) == 0 {
		add("diagram needs at least one node")
	}

	nodeIDs := map[string]bool{}
	type pt struct{ x, y float64 }
	at := map[pt]string{}
	for i, n := range d.Nodes {
		name := fmt.Sprintf("node %q", n.ID)
		if n.ID == "" {
			name = fmt.Sprintf("node #%d", i+1)
			add("%s: id is required", name)
		} else {
			if !idPattern.MatchString(n.ID) {
				add("%s: id must be lowercase letters, digits and single hyphens", name)
			}
			if nodeIDs[n.ID] {
				add("%s: duplicate node id", name)
			}
			nodeIDs[n.ID] = true
		}
		if strings.TrimSpace(n.Label) == "" {
			add("%s: label is required", name)
		}
		if !slices.Contains(NodeKinds, n.Kind) {
			add("%s: unknown kind %q (want one of: %s)", name, n.Kind, strings.Join(NodeKinds, ", "))
		}
		if d.Width > 0 && d.Height > 0 && (n.X < 0 || n.X > d.Width || n.Y < 0 || n.Y > d.Height) {
			add("%s: position (%v, %v) is outside the %vx%v canvas", name, n.X, n.Y, d.Width, d.Height)
		}
		if other, dup := at[pt{n.X, n.Y}]; dup && n.ID != "" {
			add("%s: has exactly the same position as node %q", name, other)
		} else {
			at[pt{n.X, n.Y}] = n.ID
		}
	}

	groupIDs := map[string]bool{}
	for _, g := range d.Groups {
		name := fmt.Sprintf("group %q", g.ID)
		if !idPattern.MatchString(g.ID) {
			add("%s: id must be lowercase letters, digits and single hyphens", name)
		}
		if groupIDs[g.ID] {
			add("%s: duplicate group id", name)
		}
		groupIDs[g.ID] = true
		if strings.TrimSpace(g.Label) == "" {
			add("%s: label is required", name)
		}
		for _, m := range g.Nodes {
			if !nodeIDs[m] {
				add("%s: member node %q does not exist", name, m)
			}
		}
		if g.Box != nil && (g.Box.Width <= 0 || g.Box.Height <= 0) {
			add("%s: box width and height must be positive", name)
		}
	}
	for _, n := range d.Nodes {
		if n.Group != "" && !groupIDs[n.Group] {
			add("node %q: group %q does not exist", n.ID, n.Group)
		}
	}
	for _, g := range d.Groups {
		members := len(g.Nodes)
		for _, n := range d.Nodes {
			if n.Group == g.ID {
				members++
			}
		}
		if members == 0 && g.Box == nil {
			add("group %q: needs member nodes or an explicit box", g.ID)
		}
	}

	linkIDs := map[string]bool{}
	for i, l := range d.Links {
		name := fmt.Sprintf("link %q", l.ID)
		if l.ID == "" {
			name = fmt.Sprintf("link #%d", i+1)
			add("%s: id is required", name)
		} else {
			if !idPattern.MatchString(l.ID) {
				add("%s: id must be lowercase letters, digits and single hyphens", name)
			}
			if linkIDs[l.ID] {
				add("%s: duplicate link id", name)
			}
			if nodeIDs[l.ID] {
				add("%s: id is also used by a node (node and link ids share one namespace)", name)
			}
			linkIDs[l.ID] = true
		}
		if !nodeIDs[l.From] {
			add("%s: from %q is not a node in this diagram", name, l.From)
		}
		if !nodeIDs[l.To] {
			add("%s: to %q is not a node in this diagram", name, l.To)
		}
		if l.From != "" && l.From == l.To {
			add("%s: from and to are the same node", name)
		}
		if !slices.Contains(LinkStyles, l.Style) {
			add("%s: unknown style %q (want one of: %s)", name, l.Style, strings.Join(LinkStyles, ", "))
		}
	}

	known := func(id string) bool { return nodeIDs[id] || linkIDs[id] }
	for i, s := range d.Steps {
		name := fmt.Sprintf("step %d", i+1)
		if strings.TrimSpace(s.Title) == "" {
			add("%s: title is required", name)
		}
		if strings.TrimSpace(s.Narration) == "" {
			add("%s: narration is required", name)
		}
		if len(s.Highlight)+len(s.Show)+len(s.Hide)+len(s.Packets)+len(s.Callouts) == 0 {
			add("%s: references nothing (needs highlight, show, hide, packets or callouts)", name)
		}
		for field, ids := range map[string][]string{"highlight": s.Highlight, "show": s.Show, "hide": s.Hide} {
			for _, id := range ids {
				if !known(id) {
					add("%s: %s refers to unknown node or link %q", name, field, id)
				}
			}
		}
		for j, c := range s.Callouts {
			if !known(c.Target) {
				add("%s: callout %d targets unknown node or link %q", name, j+1, c.Target)
			}
			if strings.TrimSpace(c.Text) == "" {
				add("%s: callout %d has no text", name, j+1)
			}
		}
		for j, p := range s.Packets {
			pn := fmt.Sprintf("%s: packet %d", name, j+1)
			if _, err := d.resolveRoute(p.Path); err != nil {
				add("%s: %v", pn, err)
			}
			if p.ColorStyle != "" && !slices.Contains(LinkStyles, p.ColorStyle) {
				add("%s: unknown color_style %q (want a link style)", pn, p.ColorStyle)
			}
			if p.DelayMS < 0 {
				add("%s: delay_ms must not be negative", pn)
			}
		}
	}
}

// normalized returns a copy with derived packet routes filled in and nil slices made empty,
// so JSON consumers never see null lists. The diagram must already be valid.
func (d Diagram) normalized() *Diagram {
	out := d
	out.Nodes = append([]DiagramNode{}, d.Nodes...)
	out.Groups = append([]DiagramGroup{}, d.Groups...)
	for i := range out.Groups {
		out.Groups[i].Nodes = append([]string{}, out.Groups[i].Nodes...)
	}
	out.Links = append([]DiagramLink{}, d.Links...)
	out.Steps = make([]DiagramStep, len(d.Steps))
	for i, s := range d.Steps {
		s.Highlight = append([]string{}, s.Highlight...)
		s.Show = append([]string{}, s.Show...)
		s.Hide = append([]string{}, s.Hide...)
		s.Callouts = append([]DiagramCallout{}, s.Callouts...)
		packets := make([]DiagramPacket, len(s.Packets))
		for j, p := range s.Packets {
			p.Route, _ = d.resolveRoute(p.Path)
			packets[j] = p
		}
		s.Packets = packets
		out.Steps[i] = s
	}
	return &out
}
