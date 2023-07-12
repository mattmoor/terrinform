package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
)

type message struct {
	Type string `json:"type"`
	Hook hook   `json:"hook"`
}

type hook struct {
	Action         string   `json:"action"`
	Resource       resource `json:"resource"`
	ElapsedSeconds int      `json:"elapsed_seconds"`
}

type resource struct {
	Addr            string `json:"addr"`
	ImpliedProvider string `json:"implied_provider"`
	ResourceType    string `json:"resource_type"`
	ResourceName    string `json:"resource_name"`
}

type latency struct {
	TotalTimeSeconds int
	MinTimeSeconds   int
	MaxTimeSeconds   int
	Instances        int
}

func (l *latency) add(sec int) {
	l.TotalTimeSeconds += sec
	l.Instances++
	if sec < l.MinTimeSeconds || l.Instances == 1 {
		l.MinTimeSeconds = sec
	}
	if sec > l.MaxTimeSeconds {
		l.MaxTimeSeconds = sec
	}
}

func (l latency) average() float64 {
	return float64(l.TotalTimeSeconds) / float64(l.Instances)
}

func keys(m map[string]latency) []string {
	// return the keys fromthe map
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func printTopN(dimension string, n int, m map[string]latency) {
	ks := keys(m)
	sort.Slice(ks, func(i, j int) bool {
		lhs, rhs := m[ks[i]].average(), m[ks[j]].average()
		if lhs == rhs {
			return ks[i] > ks[j]
		}
		return lhs > rhs
	})

	if n > len(ks) {
		n = len(ks)
	}

	total := 0
	for _, key := range ks {
		total += m[key].TotalTimeSeconds
	}

	fmt.Printf("Top %d by %s (total: %d sec):\n", n, dimension, total)
	for _, key := range ks[:n] {
		if m[key].Instances == 1 {
			fmt.Printf("  %s: %d sec (%.2f%%)\n", key, m[key].TotalTimeSeconds, float64(m[key].TotalTimeSeconds*100)/float64(total))
		} else {
			fmt.Printf("  %s: total %d sec (%.2f%%) over %d instances, avg: %.2f sec [%d, %d]\n", key, m[key].TotalTimeSeconds, float64(m[key].TotalTimeSeconds*100)/float64(total), m[key].Instances, m[key].average(), m[key].MinTimeSeconds, m[key].MaxTimeSeconds)
		}
	}
}

func main() {
	dec := json.NewDecoder(os.Stdin)

	byAddr := make(map[string]latency)
	byProvider := make(map[string]latency)
	byResource := make(map[string]latency)

	// Read all of the JSON objects from the stream and print them.
	for {
		var msg message
		if err := dec.Decode(&msg); err == io.EOF {
			break
		} else if err != nil {
			log.Printf("GOT: %v", err)
			break
		}
		switch msg.Type {
		case "apply_complete":
			// Process these!
		default:
			continue
		}

		// Accumulate by the full addr:
		{
			l := byAddr[msg.Hook.Resource.Addr]
			l.add(msg.Hook.ElapsedSeconds)
			byAddr[msg.Hook.Resource.Addr] = l
		}

		// Accumulate by the provider:
		{
			l := byProvider[msg.Hook.Resource.ImpliedProvider]
			l.add(msg.Hook.ElapsedSeconds)
			byProvider[msg.Hook.Resource.ImpliedProvider] = l
		}

		// Accumulate by the resource type:
		{
			l := byResource[msg.Hook.Resource.ResourceType]
			l.add(msg.Hook.ElapsedSeconds)
			byResource[msg.Hook.Resource.ResourceType] = l
		}
	}

	printTopN("address", 10, byAddr)
	printTopN("provider (avg)", 10, byProvider)
	printTopN("resource (avg)", 10, byResource)
}
