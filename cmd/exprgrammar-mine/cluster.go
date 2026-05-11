// SPDX-License-Identifier: Apache-2.0

package main

import "sort"

type Summary struct {
	bySlot map[string]*SlotSummary
}

type SlotSummary struct {
	Count   int
	Samples map[string]int
}

func Cluster(m *Miner) *Summary {
	s := &Summary{bySlot: map[string]*SlotSummary{}}
	for _, r := range m.Records {
		ss, ok := s.bySlot[r.SlotPath]
		if !ok {
			ss = &SlotSummary{Samples: map[string]int{}}
			s.bySlot[r.SlotPath] = ss
		}
		ss.Count++
		ss.Samples[r.SourceText]++
	}
	return s
}

func (s *Summary) SlotCount(slot string) int {
	if ss, ok := s.bySlot[slot]; ok {
		return ss.Count
	}
	return 0
}

func (s *Summary) SlotSamples(slot string, n int) []string {
	ss, ok := s.bySlot[slot]
	if !ok {
		return nil
	}
	type kv struct {
		k string
		v int
	}
	pairs := make([]kv, 0, len(ss.Samples))
	for k, v := range ss.Samples {
		pairs = append(pairs, kv{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].v != pairs[j].v {
			return pairs[i].v > pairs[j].v
		}
		return pairs[i].k < pairs[j].k
	})
	out := make([]string, 0, n)
	for i, p := range pairs {
		if i >= n {
			break
		}
		out = append(out, p.k)
	}
	return out
}

func (s *Summary) AllSlots() []string {
	keys := make([]string, 0, len(s.bySlot))
	for k := range s.bySlot {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
