package enterprisenumbers

import "testing"

func TestEnterpriseNumbers(t *testing.T) {
	doLookup(
		t,
		4,
		"Unix",
	)
	doLookup(
		t,
		39612,
		"Stantec Consulting",
	)
	doLookup(
		t,
		58275,
		"Eternalplanet Energy Ltd",
	)
}

func doLookup(t *testing.T, number uint32, expectedOrganization string) {
	org, ok := Lookup(number)
	if !ok {
		t.Error("failed to lookup enterprise number ")
	}
	if org != expectedOrganization {
		t.Errorf("expected organization for enterprise number %d to be %s but was: %s", number, expectedOrganization, org)
	}
}
