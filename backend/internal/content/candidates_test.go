package content

import (
	"os"
	"strings"
	"testing"
)

func TestParseCandidates(t *testing.T) {
	body := "text\n- **[observation candidate]** Open: Configure > Edges, the Device tab. Expect: HA reads None. Docs may differ: it could be a Cluster.\n- **[observation candidate]** In a lab only: Force HA Failover. Expect: an event.\nplain line\n"
	got := ParseCandidates("x", body)
	if len(got) != 2 {
		t.Fatalf("got %d candidates, want 2", len(got))
	}
	if got[0].Screen != "Configure > Edges, the Device tab." || got[0].Expect != "HA reads None." || got[0].Differ != "it could be a Cluster." {
		t.Errorf("first candidate parsed wrong: %+v", got[0])
	}
	if got[1].Screen != "In a lab only: Force HA Failover." || got[1].Expect != "an event." || got[1].Differ != "" {
		t.Errorf("second candidate parsed wrong: %+v", got[1])
	}
}

// Every lesson of Sections 7, 8, 9 and 10 ends with a "Check this in your own VCO" box listing 3 to 5 candidates, each with a screen, a
// documented expectation and where the documentation may differ.
func TestLessonsListObservationCandidates(t *testing.T) {
	c, err := Load(os.DirFS("../../../content"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{"topologies": 7, "network-services-config": 8, "profile-policy-config": 7, "troubleshooting": 10}
	got := map[string]int{}
	for _, s := range c.Sections {
		if want[s.Meta.ID] == 0 {
			continue
		}
		for _, l := range s.Lessons {
			got[s.Meta.ID]++
			if !strings.Contains(l.Body, "Check this in your own VCO") {
				t.Errorf("%s has no \"Check this in your own VCO\" box", l.Meta.ID)
			}
			cands := ParseCandidates(l.Meta.ID, l.Body)
			if len(cands) < 3 || len(cands) > 5 {
				t.Errorf("%s lists %d observation candidates, want 3 to 5", l.Meta.ID, len(cands))
			}
			for _, cd := range cands {
				if cd.Screen == "" || cd.Expect == "" || cd.Differ == "" {
					t.Errorf("%s: candidate %+v needs a screen (Open:), an Expect: and a Docs may differ:", l.Meta.ID, cd)
				}
			}
		}
	}
	for id, n := range want {
		if got[id] != n {
			t.Errorf("section %s: %d lessons found, want %d", id, got[id], n)
		}
	}
}
