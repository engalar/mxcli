// SPDX-License-Identifier: Apache-2.0

package executor

import "testing"

// TestAttrNameForOData_ReservedWords verifies that Mendix-reserved attribute
// names are prefixed with the entity name so Studio Pro does not reject them.
// Regression test for issue #526.
func TestAttrNameForOData_ReservedWords(t *testing.T) {
	cases := []struct {
		prop   string
		entity string
		want   string
	}{
		// Already-covered names
		{"Id", "Photo", "PhotoId"},
		{"id", "Photo", "Photoid"},
		{"Name", "Airline", "AirlineName"},
		{"name", "Airline", "Airlinename"},
		// Newly-added reserved names (issue #526)
		{"Owner", "Trip", "TripOwner"},
		{"owner", "Trip", "Tripowner"},
		{"Type", "Flight", "FlightType"},
		{"type", "Flight", "Flighttype"},
		{"Context", "Person", "PersonContext"},
		{"context", "Person", "Personcontext"},
		{"ChangedBy", "Event", "EventChangedBy"},
		{"changedby", "Event", "Eventchangedby"},
		{"ChangedDate", "Event", "EventChangedDate"},
		{"changeddate", "Event", "Eventchangeddate"},
		{"CreatedDate", "Event", "EventCreatedDate"},
		{"createddate", "Event", "Eventcreateddate"},
		// Non-reserved names must pass through unchanged
		{"AirlineCode", "Airline", "AirlineCode"},
		{"Concurrency", "Airline", "Concurrency"},
		{"FirstName", "Person", "FirstName"},
	}

	for _, tc := range cases {
		got := attrNameForOData(tc.prop, tc.entity)
		if got != tc.want {
			t.Errorf("attrNameForOData(%q, %q) = %q; want %q", tc.prop, tc.entity, got, tc.want)
		}
	}
}
