// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

type settingsBackend struct {
	reader *modelsdkmpr.Reader
}

func newSettingsBackend(reader *modelsdkmpr.Reader) *settingsBackend {
	return &settingsBackend{reader: reader}
}

func (b *settingsBackend) GetProjectSettings() (*model.ProjectSettings, error) {
	rawUnits, err := b.reader.ListRawUnitsByType(projectSettingsBsonType)
	if err != nil {
		return nil, err
	}
	if len(rawUnits) == 0 {
		return nil, fmt.Errorf("project settings not found")
	}
	ru := rawUnits[0]
	return parseProjectSettingsRaw(string(ru.ID), string(ru.ContainerID), ru.Contents)
}
