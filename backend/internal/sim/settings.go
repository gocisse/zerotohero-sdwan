package sim

import (
	"fmt"
	"sort"
)

// ValueKind is the shape of a setting's value.
type ValueKind string

const (
	KindBool   ValueKind = "bool"
	KindScalar ValueKind = "scalar" // bool, number or string: opaque to the engine (assumption 33)
)

// Device tab sections the fact sheets confirm, and the catch-all for the rest.
const (
	SectionConnectivity = "Connectivity"
	SectionRouting      = "Routing & NAT"
	SectionFirewall     = "Firewall"
	SectionOther        = "Other (section not confirmed)"
)

// SettingDef describes one setting the simulator knows. Only settings whose Override is documented are listed
// (the complete overridable list is not documented: rule nm-override-list).
type SettingDef struct {
	ID      string
	Label   string
	Section string
	Kind    ValueKind
	// SectionKnown is false when the section was not read (assumption 36).
	SectionKnown bool
	// Default is the documented default; HasDefault false means "no documented default" (nm-setting-defaults).
	Default     any
	HasDefault  bool
	DefaultRule string
}

// Settings is the registry of device settings (profile level, overridable on an Edge).
var Settings = []SettingDef{
	{ID: "connectivity.arp_timeouts", Label: "ARP Timeouts", Section: SectionConnectivity, Kind: KindScalar, SectionKnown: true},
	{ID: "routing.bfd", Label: "BFD", Section: SectionRouting, Kind: KindScalar, SectionKnown: true},
	{ID: "routing.ecmp", Label: "ECMP", Section: SectionRouting, Kind: KindScalar, SectionKnown: true},
	{ID: "connectivity.wifi_radio", Label: "Wi-Fi Radio", Section: SectionOther, Kind: KindScalar},
	{ID: "connectivity.global_ipv6", Label: "Global IPv6", Section: SectionOther, Kind: KindScalar},
	{ID: "services.snmp", Label: "SNMP", Section: SectionOther, Kind: KindScalar},
	{ID: "services.radius", Label: "RADIUS", Section: SectionOther, Kind: KindScalar},
	// rule: fw-stateful-default
	{ID: "firewall.stateful_firewall", Label: "Stateful Firewall", Section: SectionFirewall, Kind: KindBool, SectionKnown: true, Default: false, HasDefault: true, DefaultRule: RFwStatefulDefault},
}

// StatefulSetting is the id of the Stateful Firewall setting (assumption 39: a Profile setting an Edge may override).
const StatefulSetting = "firewall.stateful_firewall"

var settingByID = func() map[string]SettingDef {
	m := map[string]SettingDef{}
	for _, s := range Settings {
		m[s.ID] = s
	}
	return m
}()

// SettingByID looks a setting up.
func SettingByID(id string) (SettingDef, bool) { s, ok := settingByID[id]; return s, ok }

// SettingIDs lists the registry ids, sorted (for error messages).
func SettingIDs() []string {
	ids := make([]string, 0, len(Settings))
	for _, s := range Settings {
		ids = append(ids, s.ID)
	}
	sort.Strings(ids)
	return ids
}

// checkValue reports why v is not a valid value for the setting, or "".
func (s SettingDef) checkValue(v any) string {
	switch v.(type) {
	case bool:
		return ""
	case int, int64, float64, string:
		if s.Kind == KindBool {
			return fmt.Sprintf("%s takes true or false", s.Label)
		}
		return ""
	case nil:
		return "value is empty"
	default:
		return fmt.Sprintf("%s takes a single value (true, false, a number or text), not a list or map", s.Label)
	}
}
