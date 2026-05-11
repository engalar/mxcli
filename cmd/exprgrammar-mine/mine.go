// SPDX-License-Identifier: Apache-2.0

package main

type SlotRecord struct {
	SlotPath   string
	SourceText string
	Microflow  string
}

type Miner struct {
	Records []SlotRecord
}

func NewMiner() *Miner {
	return &Miner{Records: []SlotRecord{}}
}
