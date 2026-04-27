package printer

import (
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/core"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model"
)

func (bp *basePrinter) commandToSubRows(node *core.SeqNode) []model.SubRow {
	srs := bp.config.RowMapCmd(node.Cmd, bp.config.WayCount)
	for i := range srs {
		srs[i].SourceLineNum = node.LineNum
	}
	return srs
}

func expandSubRowGroup(sr model.SubRow, status model.SubRowStatus) []model.SubRow {
	macros := sr.EffectiveMacros()
	expanded := make([]model.SubRow, len(macros))
	for i, m := range macros {
		expanded[i] = model.SubRow{
			Macro:         m,
			Addr:          sr.Addr,
			Settings:      sr.Settings,
			SF:            sr.SF,
			XYC:           sr.XYC,
			Timing:        sr.Timing,
			TimingGroup:   sr.TimingGroup,
			Cmd:           sr.Cmd,
			Status:        status,
			Violation:     sr.Violation,
			RepeatCnt:     0,
			SourceLineNum: sr.SourceLineNum,
		}
	}
	return expanded
}
