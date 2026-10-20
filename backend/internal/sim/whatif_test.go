package sim

import (
	"encoding/json"
	"strings"
	"testing"
)

func diffFor(t *testing.T, res *WhatIfResult, edge string) EdgeDiff {
	t.Helper()
	for _, d := range res.Edges {
		if d.EdgeID == edge {
			return d
		}
	}
	t.Fatalf("no diff for %s", edge)
	return EdgeDiff{}
}

func hasItem(d EdgeDiff, key string) bool {
	for _, it := range d.Items {
		if it.Key == key {
			return true
		}
	}
	return false
}

func TestProfileChangeReachesOnlyNonOverridingEdges(t *testing.T) {
	tp := baseTopology()
	res, err := WhatIf(tp, Change{Type: ChSetProfileSetting, Profile: "p1", Setting: "connectivity.arp_timeouts", Value: 999})
	if err != nil {
		t.Fatal(err)
	}
	e1, e2 := diffFor(t, res, "e1"), diffFor(t, res, "e2")
	if e1.Status != "kept-override" || len(e1.Items) != 0 {
		t.Errorf("e1 overrides ARP timeouts and keeps its value: %+v", e1)
	}
	foundA5 := false
	for _, n := range e1.Notes {
		foundA5 = foundA5 || (n.Basis.Rule == RA5ProfileEditSkips && n.Basis.Status == StatusAssumption)
	}
	if !foundA5 {
		t.Errorf("the kept-override outcome rests on assumption 5 and must say so: %+v", e1.Notes)
	}
	if e2.Status != "changed" || !hasItem(e2, "device:connectivity.arp_timeouts") {
		t.Errorf("e2 inherits and changes: %+v", e2)
	}
	if v := e2.Items[0]; v.Before != float64(120) && v.Before != 120 {
		t.Errorf("before value: %v", v.Before)
	}
}

func TestProfileChangeLeavesOtherProfilesAlone(t *testing.T) {
	tp := baseTopology()
	tp.Sites[0].Edges[1].Profile = "p2"
	res, err := WhatIf(tp, Change{Type: ChSetProfileSetting, Profile: "p1", Setting: "routing.bfd", Value: true})
	if err != nil {
		t.Fatal(err)
	}
	if d := diffFor(t, res, "e2"); d.Status != "unchanged" {
		t.Errorf("e2 is on p2: %+v", d)
	}
}

func TestRemoveEdgeOverrideFallsBackToProfile(t *testing.T) {
	res, err := WhatIf(baseTopology(), Change{Type: ChRemoveEdgeOverride, Edge: "e1", Setting: "connectivity.arp_timeouts"})
	if err != nil {
		t.Fatal(err)
	}
	d := diffFor(t, res, "e1")
	if d.Status != "changed" || len(d.Items) != 1 {
		t.Fatalf("%+v", d)
	}
	it := d.Items[0]
	if it.AfterSource == nil || it.AfterSource.Kind != SrcProfile || it.BeforeSource.Kind != SrcEdgeOverride {
		t.Errorf("source should move from override to profile: %+v", it)
	}
	if len(d.Notes) == 0 || d.Notes[0].Basis.Rule != RA8RevertToProfile {
		t.Errorf("revert is assumption 8: %+v", d.Notes)
	}
	if _, err := WhatIf(baseTopology(), Change{Type: ChRemoveEdgeOverride, Edge: "e2", Setting: "routing.bfd"}); err == nil {
		t.Error("removing an override that does not exist must be an error")
	}
}

// JSON arrays must never be null: the UI reads .length on them.
func TestResultListsAreNeverNil(t *testing.T) {
	res, err := WhatIf(baseTopology(), Change{Type: ChSetProfileSetting, Profile: "p1", Setting: "routing.bfd", Value: true})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(res)
	for _, bad := range []string{`"edges":null`, `"issues":null`, `"not_modelled":null`, `"items":null`, `"notes":null`} {
		if strings.Contains(string(data), bad) {
			t.Errorf("result contains %s: %s", bad, data)
		}
	}
}

func TestAddEdgeOverride(t *testing.T) {
	res, err := WhatIf(baseTopology(), Change{Type: ChSetEdgeOverride, Edge: "e2", Setting: "routing.bfd", Value: true})
	if err != nil {
		t.Fatal(err)
	}
	d := diffFor(t, res, "e2")
	if d.Status != "changed" || d.Items[0].AfterSource.Kind != SrcEdgeOverride {
		t.Errorf("%+v", d)
	}
	if diffFor(t, res, "e1").Status != "unchanged" {
		t.Error("other Edges are unaffected")
	}
}

func TestProfileSwitchSideEffects(t *testing.T) {
	tp := baseTopology()
	res, err := WhatIf(tp, Change{Type: ChAssignProfile, Edge: "e1", Profile: "p2"})
	if err != nil {
		t.Fatal(err)
	}
	d := diffFor(t, res, "e1")
	if d.Profile != "p2" || d.Status != "changed" {
		t.Fatalf("%+v", d)
	}
	if !hasItem(d, "vlan:20") {
		t.Errorf("VLAN 20 was inherited from p1 and p2 does not define it, so it is removed: %+v", d.Items)
	}
	if !hasItem(d, "vlan:30") {
		t.Errorf("VLAN 30 is inherited from p2: %+v", d.Items)
	}
	for _, it := range d.Items {
		if it.Key == "vlan:10" && (jsonKey(it.Before) != jsonKey(it.After) || it.AfterSource.ID != "p2") {
			t.Errorf("VLAN 10 exists in both Profiles and is matched by id; only its source changes: %+v", it)
		}
		if strings.HasPrefix(it.Key, "interface:") && it.Key != "interface:ge4" {
			t.Errorf("only the reverted interface should differ: %+v", it)
		}
	}
	if len(d.Reverted) != 1 || d.Reverted[0] != "ge4" {
		t.Errorf("ge4 was tied to removed VLAN 20 (and overridden) so it reverts: %v", d.Reverted)
	}
	var ge4 *DiffItem
	for i := range d.Items {
		if d.Items[i].Key == "interface:ge4" {
			ge4 = &d.Items[i]
		}
	}
	if ge4 == nil {
		t.Fatal("ge4 should differ")
	}
	after, _ := json.Marshal(ge4.After)
	if !strings.Contains(string(after), `"override":false`) || strings.Contains(string(after), `"vlan":20`) {
		t.Errorf("ge4 should have lost its VLAN and its override: %s", after)
	}
	rules := map[string]Basis{}
	for _, n := range d.Notes {
		rules[n.Basis.Rule] = n.Basis
	}
	if rules[RSwitchVlans].Status != StatusVerified {
		t.Errorf("VLAN removal is verified: %v", rules)
	}
	if rules[RNMSwitchBGP].Status != StatusNotModelled || rules[RNMSwitchOverrides].Status != StatusNotModelled {
		t.Errorf("BGP filters and other overrides must be marked not modelled: %v", rules)
	}
	if rules[RA35SwitchVlanByID].Status != StatusAssumption {
		t.Errorf("matching VLANs by id is assumption 35: %v", rules)
	}
	found := false
	for _, nm := range res.NotModelled {
		found = found || nm.Rule == RNMSwitchBGP
	}
	if !found {
		t.Error("result-level not-modelled list should name the BGP side effect")
	}
	if diffFor(t, res, "e2").Status != "unchanged" {
		t.Error("e2 does not change")
	}
	for _, is := range res.Issues {
		if strings.Contains(is.Path, "ge4") {
			t.Errorf("the reverted interface is a documented effect, not an authoring issue: %v", is)
		}
	}
}

func TestProfileSwitchReportsIntroducedIssues(t *testing.T) {
	tp := baseTopology()
	tp.Profiles[1].Segments = []string{"global"} // p2 lacks the guest segment e1 uses
	tp.Profiles[0].BusinessRules = tp.Profiles[0].BusinessRules[:1]
	res, err := WhatIf(tp, Change{Type: ChAssignProfile, Edge: "e1", Profile: "p2"})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, is := range res.Issues {
		found = found || strings.Contains(is.Message, "assumption 12")
	}
	if !found {
		t.Errorf("interface segment no longer in the Profile should be reported: %v", res.Issues)
	}
}

func TestWhatIfDoesNotMutateInput(t *testing.T) {
	tp := baseTopology()
	before, _ := json.Marshal(tp)
	for _, ch := range []Change{
		{Type: ChSetProfileSetting, Profile: "p1", Setting: "routing.bfd", Value: true},
		{Type: ChRemoveEdgeOverride, Edge: "e1", Setting: "connectivity.arp_timeouts"},
		{Type: ChAssignProfile, Edge: "e1", Profile: "p2"},
	} {
		if _, err := WhatIf(tp, ch); err != nil {
			t.Fatal(err)
		}
	}
	after, _ := json.Marshal(tp)
	if string(before) != string(after) {
		t.Fatal("WhatIf must not modify the topology")
	}
}

func TestInvalidChanges(t *testing.T) {
	cases := []Change{
		{Type: "explode"},
		{Type: ChSetProfileSetting, Profile: "nope", Setting: "routing.bfd", Value: true},
		{Type: ChSetProfileSetting, Profile: "p1", Setting: "no.such", Value: true},
		{Type: ChSetProfileSetting, Profile: "p1", Setting: StatefulSetting, Value: "yes"},
		{Type: ChSetEdgeOverride, Edge: "nope", Setting: "routing.bfd", Value: true},
		{Type: ChAssignProfile, Edge: "e1", Profile: "nope"},
		{Type: ChAssignProfile, Edge: "e1", Profile: "p1"},
	}
	for _, c := range cases {
		_, err := WhatIf(baseTopology(), c)
		var ce *ChangeError
		if err == nil {
			t.Errorf("%+v should fail", c)
		} else if !asChangeError(err, &ce) {
			t.Errorf("%+v: want a ChangeError, got %T %v", c, err, err)
		}
	}
}

func asChangeError(err error, target **ChangeError) bool {
	if ce, ok := err.(*ChangeError); ok {
		*target = ce
		return true
	}
	return false
}

func TestRemoveProfileSetting(t *testing.T) {
	res, err := WhatIf(baseTopology(), Change{Type: ChRemoveProfileSetting, Profile: "p1", Setting: "connectivity.arp_timeouts"})
	if err != nil {
		t.Fatal(err)
	}
	d := diffFor(t, res, "e2")
	if d.Status != "changed" || d.Items[0].AfterSource.Kind != SrcUnset {
		t.Errorf("removing the only value leaves the setting unset: %+v", d)
	}
}
