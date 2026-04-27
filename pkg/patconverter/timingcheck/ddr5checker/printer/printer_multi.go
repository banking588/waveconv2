package printer

import (
	"common/log"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/core"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model"
)

type MultiWayStrategy struct {
	loopStrategy LoopStrategy
}

func NewMultiWayStrategy() *MultiWayStrategy {
	return &MultiWayStrategy{
		loopStrategy: NewDefaultLoopStrategy(),
	}
}

func (s *MultiWayStrategy) ShouldPad() bool                 { return true }
func (s *MultiWayStrategy) SetLoopStrategy(ls LoopStrategy) { s.loopStrategy = ls }

func (s *MultiWayStrategy) ProcessCommand(nodes []*core.SeqNode, bp *basePrinter, acc *accumulator, i int, cmdIndex int) (int, int) {
	node := nodes[i]
	srs := bp.commandToSubRows(node)

	if len(srs) > 0 && srs[0].RepeatCnt > 1 {
		s.processRepeatCommand(bp, acc, &srs[0], cmdIndex)
		return i + 1, cmdIndex + 1
	}

	for _, sr := range srs {
		acc.addEntry(subRowEntry{sr: sr, cmdIndex: cmdIndex})
	}
	return i + 1, cmdIndex + 1
}

func (s *MultiWayStrategy) processRepeatCommand(bp *basePrinter, acc *accumulator, normalSR *model.SubRow, cmdIndex int) {
	wc := bp.effectiveWayCount()
	groupSize := normalSR.GroupSize()
	repeatCnt := normalSR.RepeatCnt
	groupsPerWay := wc / groupSize

	// DES 拆分吸收
	if acc.count > 0 && acc.count < wc {
		canFit := (wc - acc.count) / groupSize
		need := canFit
		if need > repeatCnt {
			need = repeatCnt
		}

		for r := 0; r < need; r++ {
			expanded := expandSubRowGroup(*normalSR, model.SubRowStatusSplitByDes)
			for _, sr := range expanded {
				acc.addEntry(subRowEntry{sr: sr, cmdIndex: cmdIndex})
			}
		}

		repeatCnt -= need
		if repeatCnt <= 0 {
			return
		}
		normalSR.RepeatCnt = repeatCnt

	} else if acc.count >= wc {
		acc.flush()
	}

	threshold := bp.config.RepeatCntOffset * groupsPerWay

	if repeatCnt <= threshold {
		for r := 0; r < repeatCnt; r++ {
			expanded := expandSubRowGroup(*normalSR, model.SubRowStatusSplitByDes)
			for _, sr := range expanded {
				acc.addEntry(subRowEntry{sr: sr, cmdIndex: cmdIndex})
			}
		}
		return
	}

	acc.flush()

	adjusted := repeatCnt - threshold
	if adjusted <= 0 {
		log.Fatalf("Command RepeatCnt (%d) - threshold (%d) = %d, must be >= 1",
			repeatCnt, threshold, adjusted)
	}

	quotient := adjusted / groupsPerWay
	remainder := adjusted % groupsPerWay

	if quotient > 0 {
		idxiSRs := expandSubRowGroup(*normalSR, model.SubRowStatusNormal)
		ctx := NewLoopContext(bp, acc, nil, "")
		s.loopStrategy.FlushAsIDXI(ctx, idxiSRs, quotient)
	}

	if remainder > 0 {
		for r := 0; r < remainder; r++ {
			expanded := expandSubRowGroup(*normalSR, model.SubRowStatusSplitByDes)
			for _, sr := range expanded {
				acc.addEntry(subRowEntry{sr: sr, cmdIndex: cmdIndex})
			}
		}
	}
}

func (s *MultiWayStrategy) ProcessLoop(nodes []*core.SeqNode, bp *basePrinter, acc *accumulator, i int, mergedLabel string) int {
	node := nodes[i]
	acc.flush()

	cmdCount := bp.countDirectCommandSubRows(node.Children)
	wc := bp.effectiveWayCount()

	if wc > 1 && cmdCount%2 != 0 {
		log.Fatalf("Loop at line %d: expanded command subrow count (%d) is not divisible by 2",
			node.LineNum, cmdCount)
	}

	ctx := NewLoopContext(bp, acc, node, mergedLabel)
	s.loopStrategy.ProcessLoop(ctx)

	return i + 1
}
