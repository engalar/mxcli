// SPDX-License-Identifier: Apache-2.0

package main

import "testing"

func TestNewMiner(t *testing.T) {
	m := NewMiner()
	if m == nil {
		t.Fatal("NewMiner returned nil")
	}
	if m.Records == nil {
		t.Fatal("Miner.Records is nil — must be allocated")
	}
}
