// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/model"
)

func TestShowSettings_Mock(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		GetProjectSettingsFunc: func() (*model.ProjectSettings, error) {
			return &model.ProjectSettings{
				Model: &model.ModelSettings{
					HashAlgorithm: "BCrypt",
					JavaVersion:   "17",
				},
			}, nil
		},
	}
	ctx, buf := newMockCtx(t, withBackend(mb))
	assertNoError(t, listSettings(ctx))

	out := buf.String()
	assertContainsStr(t, out, "Section")
	assertContainsStr(t, out, "Key Values")
}

func TestDescribeSettings_Mock(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		GetProjectSettingsFunc: func() (*model.ProjectSettings, error) {
			return &model.ProjectSettings{
				Model: &model.ModelSettings{
					HashAlgorithm: "BCrypt",
					JavaVersion:   "17",
					RoundingMode:  "HalfUp",
				},
			}, nil
		},
	}
	ctx, buf := newMockCtx(t, withBackend(mb))
	assertNoError(t, describeSettings(ctx))
	assertContainsStr(t, buf.String(), "alter settings")
}

func TestShowSettings_NotConnected(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return false },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listSettings(ctx))
}

func TestDescribeSettings_NotConnected(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return false },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, describeSettings(ctx))
}

func TestShowSettings_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		GetProjectSettingsFunc: func() (*model.ProjectSettings, error) {
			return nil, fmt.Errorf("connection lost")
		},
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listSettings(ctx))
}

func TestShowSettings_JSON(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		GetProjectSettingsFunc: func() (*model.ProjectSettings, error) {
			return &model.ProjectSettings{
				Model: &model.ModelSettings{
					HashAlgorithm: "BCrypt",
					JavaVersion:   "17",
				},
			}, nil
		},
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON))
	assertNoError(t, listSettings(ctx))
	assertValidJSON(t, buf.String())
}

func TestDescribeSettings_Languages(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		GetProjectSettingsFunc: func() (*model.ProjectSettings, error) {
			return &model.ProjectSettings{
				Language: &model.LanguageSettings{
					DefaultLanguageCode: "en_US",
					Languages: []model.Language{
						{Code: "en_US"},
						{Code: "zh_CN"},
						{Code: "nl_NL", CheckCompleteness: true},
					},
				},
			}, nil
		},
	}
	ctx, buf := newMockCtx(t, withBackend(mb))
	assertNoError(t, describeSettings(ctx))
	out := buf.String()

	assertContainsStr(t, out, "alter settings language add 'zh_CN';")
	assertContainsStr(t, out, "alter settings language add 'nl_NL' (checkCompleteness: true);")
	if strings.Contains(out, "alter settings language add 'en_US'") {
		t.Errorf("default language must not appear as add statement, got:\n%s", out)
	}
	assertContainsStr(t, out, "DefaultLanguageCode = 'en_US'")
}
