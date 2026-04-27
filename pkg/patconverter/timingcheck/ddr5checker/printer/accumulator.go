package printer

import (
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model"
)

type subRowEntry struct {
	sr       model.SubRow
	cmdIndex int
}

type pendingAssignTag struct {
	addr    string
	lineNum int
}

type accumulator struct {
	bp      *basePrinter
	entries []subRowEntry
	count   int
	result  []*model.PrintNode

	pendingLabel           string
	pendingAssignTags      []pendingAssignTag
	pendingIncrementalTags []pendingAssignTag
}

func (acc *accumulator) flush() {
	if len(acc.entries) == 0 {
		return
	}

	wc := acc.bp.effectiveWayCount()

	pn := model.NewPrintNode(model.NodeHeader{
		Operator:  "NOP",
		Interrupt: true,
	}, wc)

	if acc.pendingLabel != "" {
		pn.Header.Label = acc.pendingLabel
		acc.pendingLabel = ""
	}

	normalCount := 0
	for _, e := range acc.entries {
		pn.SubRows = append(pn.SubRows, e.sr)
		if e.sr.Violation == nil {
			normalCount++
		}
	}

	acc.bp.padSubRows(pn, normalCount)

	acc.result = append(acc.result, pn)
	acc.entries = nil
	acc.count = 0
}
func (acc *accumulator) addEntry(entry subRowEntry) {
	if entry.sr.Violation != nil {
		acc.entries = append(acc.entries, entry)
		return
	}

	wc := acc.bp.effectiveWayCount()
	if wc > 0 && acc.count >= wc {
		acc.flush()
	}

	// 指定型
	if len(acc.pendingAssignTags) > 0 {
		for _, t := range acc.pendingAssignTags {
			entry.sr.Addr = append(entry.sr.Addr, t.addr)
		}
		acc.pendingAssignTags = nil
	}

	// 累加型
	if len(acc.pendingIncrementalTags) > 0 {
		for _, t := range acc.pendingIncrementalTags {
			entry.sr.Addr = append(entry.sr.Addr, t.addr)
		}
		acc.pendingIncrementalTags = nil
	}

	acc.entries = append(acc.entries, entry)
	acc.count++
}

func (acc *accumulator) pendingAssignTagLines() []int {
	lines := make([]int, len(acc.pendingAssignTags))
	for i, t := range acc.pendingAssignTags {
		lines[i] = t.lineNum
	}
	return lines
}

func (acc *accumulator) pendingIncrementalTagLines() []int {
	lines := make([]int, len(acc.pendingIncrementalTags))
	for i, t := range acc.pendingIncrementalTags {
		lines[i] = t.lineNum
	}
	return lines
}
