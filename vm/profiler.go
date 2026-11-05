package vm

import (
	"bramandscript/code"
	"fmt"
	"sort"
	"strings"
	"time"
)

type OpcodeStat struct {
	Opcode   code.Opcode
	Name     string
	Count    uint64
	Duration time.Duration
	Percent  float64
}

type Profiler struct {
	StartTime         time.Time
	EndTime           time.Time
	TotalInstructions uint64
	OpcodeCounts      [256]uint64
	OpcodeDurations   [256]time.Duration
}

func NewProfiler() *Profiler {
	return &Profiler{}
}

func (p *Profiler) RecordOpcode(op code.Opcode, d time.Duration) {
	p.OpcodeCounts[op]++
	p.OpcodeDurations[op] += d
	p.TotalInstructions++
}

type ProfileReport struct {
	TotalDuration     time.Duration
	TotalInstructions uint64
	Stats             []OpcodeStat
}

func (p *Profiler) GenerateReport() *ProfileReport {
	totalDur := p.EndTime.Sub(p.StartTime)
	if totalDur <= 0 {
		var sumDur time.Duration
		for _, d := range p.OpcodeDurations {
			sumDur += d
		}
		totalDur = sumDur
	}

	var stats []OpcodeStat
	var totalMeasured time.Duration
	for i := 0; i < 256; i++ {
		count := p.OpcodeCounts[i]
		if count > 0 {
			dur := p.OpcodeDurations[i]
			totalMeasured += dur
			op := code.Opcode(i)
			name := fmt.Sprintf("0x%02X", i)
			def, err := code.Lookup(byte(op))
			if err == nil && def != nil {
				name = def.Name
			}
			stats = append(stats, OpcodeStat{
				Opcode:   op,
				Name:     name,
				Count:    count,
				Duration: dur,
			})
		}
	}

	basis := totalMeasured
	if basis <= 0 {
		basis = totalDur
	}

	for i := range stats {
		if basis > 0 {
			stats[i].Percent = float64(stats[i].Duration) / float64(basis) * 100.0
		}
	}

	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Duration == stats[j].Duration {
			return stats[i].Count > stats[j].Count
		}
		return stats[i].Duration > stats[j].Duration
	})

	return &ProfileReport{
		TotalDuration:     totalDur,
		TotalInstructions: p.TotalInstructions,
		Stats:             stats,
	}
}

func (pr *ProfileReport) String() string {
	var sb strings.Builder
	sb.WriteString("\n✦ BramandScript Execution Profiler\n")
	sb.WriteString("────────────────────────────────────────────────────────────────────────────\n")
	sb.WriteString(fmt.Sprintf("%-22s %8s %14s %12s   %-16s\n", "OPCODE", "COUNT", "TOTAL TIME", "AVG/OP", "HOTSPOT PROFILE"))
	sb.WriteString("────────────────────────────────────────────────────────────────────────────\n")

	for _, s := range pr.Stats {
		avg := time.Duration(0)
		if s.Count > 0 {
			avg = s.Duration / time.Duration(s.Count)
		}
		barLen := int((s.Percent / 100.0) * 12.0)
		if barLen < 1 && s.Percent > 1.0 {
			barLen = 1
		}
		if barLen > 12 {
			barLen = 12
		}
		bar := strings.Repeat("█", barLen) + strings.Repeat("░", 12-barLen)
		sb.WriteString(fmt.Sprintf("%-22s %8d %14s %12s   %s %5.1f%%\n",
			s.Name, s.Count, formatDuration(s.Duration), formatDuration(avg), bar, s.Percent))
	}

	sb.WriteString("────────────────────────────────────────────────────────────────────────────\n")
	ips := float64(0)
	if pr.TotalDuration.Seconds() > 0 {
		ips = float64(pr.TotalInstructions) / pr.TotalDuration.Seconds()
	}
	sb.WriteString(fmt.Sprintf("Total Instructions: %d | Execution Time: %s | Speed: %.1f KIPS\n\n",
		pr.TotalInstructions, formatDuration(pr.TotalDuration), ips/1000.0))
	return sb.String()
}

func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%d ns", d.Nanoseconds())
	} else if d < time.Millisecond {
		return fmt.Sprintf("%.2f µs", float64(d.Nanoseconds())/1000.0)
	} else if d < time.Second {
		return fmt.Sprintf("%.2f ms", float64(d.Microseconds())/1000.0)
	}
	return fmt.Sprintf("%.3f s", d.Seconds())
}
