package workflows

import "github.com/mendixlabs/mxcli/modelsdk/element"

// InsertActivitiesAt inserts an activity at the given index in the Flow's activities list.
func (o *Flow) InsertActivitiesAt(index int, v element.Element) {
	o.activities.InsertAt(index, v)
}
