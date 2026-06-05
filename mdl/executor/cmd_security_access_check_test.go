// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"
)

func TestAccessGap_SuggestedMDL_EntityRead(t *testing.T) {
	g := AccessGap{
		GapType:    GapEntityRead,
		ModuleRole: "HD.CustomerRole",
		EntityQN:   "HD.UserProfile",
	}
	want := "grant HD.CustomerRole on HD.UserProfile (read *);"
	if got := g.SuggestedMDL(); got != want {
		t.Errorf("SuggestedMDL() = %q, want %q", got, want)
	}
}

func TestAccessGap_SuggestedMDL_EntityWrite(t *testing.T) {
	g := AccessGap{
		GapType:    GapEntityWrite,
		ModuleRole: "HD.CustomerRole",
		EntityQN:   "HD.PasswordForm",
	}
	want := "grant HD.CustomerRole on HD.PasswordForm (create, read *, write *);"
	if got := g.SuggestedMDL(); got != want {
		t.Errorf("SuggestedMDL() = %q, want %q", got, want)
	}
}

func TestAccessGap_SuggestedMDL_MFExecute(t *testing.T) {
	g := AccessGap{
		GapType: GapMFExecute,
		ModuleRole: "HD.CustomerRole",
		MFQN:       "HD.DS_GetMyProfile",
	}
	want := "grant execute on microflow HD.DS_GetMyProfile to HD.CustomerRole;"
	if got := g.SuggestedMDL(); got != want {
		t.Errorf("SuggestedMDL() = %q, want %q", got, want)
	}
}
