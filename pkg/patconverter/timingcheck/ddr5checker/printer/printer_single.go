package printer

import (
	"fmt"

	"common/log"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/core"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model"
	"waveconv/pkg/util"
)

type SingleWayStrategy struct {
	loopStrategy LoopStrategy
}

func NewSingleWayStrategy() *SingleWayStrategy {
	return &SingleWayStrategy{
		loopStrategy: NewSingleLoopStrategy(),
	}
}

func (s *SingleWayStrategy) ShouldPad() bool                 { return false }
func (s *SingleWayStrategy) SetLoopStrategy(ls LoopStrategy) { s.loopStrategy = ls }
func (s *SingleWayStrategy) ProcessCommand(nodes []*core.SeqNode, bp *basePrinter, acc *accumulator, i int, cmdIndex int) (int, int) {
	node := nodes[i]
	srs := bp.commandToSubRows(node)

	if len(srs) > 0 && srs[0].RepeatCnt > 1 {
		s.processRepeatCommand(bp, acc, &srs[0], cmdIndex)
		return i + 1, cmdIndex + 1
	}

	// 沒有 RepeatCnt
	if len(srs) > 0 && len(srs[0].Macros) > 0 {
		// 情況 2A：沒有 RepeatCnt，有 Macros 陣列 → 按macros的數量展開
		expanded := expandSubRowGroup(srs[0], model.SubRowStatusNormal)
		for _, sr := range expanded {
			acc.addEntry(subRowEntry{sr: sr, cmdIndex: cmdIndex})
		}
		return i + 1, cmdIndex + 1
	}

	// 情況 2B：沒有 RepeatCnt，沒有 Macros → 原邏輯, 直接打印就行
	for _, sr := range srs {
		acc.addEntry(subRowEntry{sr: sr, cmdIndex: cmdIndex})
	}
	return i + 1, cmdIndex + 1
}

func (s *SingleWayStrategy) processRepeatCommand(bp *basePrinter, acc *accumulator, normalSR *model.SubRow, cmdIndex int) {
	// 情況 1A：有 RepeatCnt，有 Macros 陣列
	if len(normalSR.Macros) > 0 {
		s.processRepeatMacrosCommand(bp, acc, normalSR, cmdIndex)
		return
	}

	// 情況 1B：有 RepeatCnt，沒有 Macros → 原邏輯, 直接打印就行
	groupSize := normalSR.GroupSize()
	repeatCnt := normalSR.RepeatCnt

	wc := bp.effectiveWayCount()
	if wc <= 0 || wc < groupSize {
		for r := 0; r < repeatCnt; r++ {
			expanded := expandSubRowGroup(*normalSR, model.SubRowStatusNormal)
			for _, sr := range expanded {
				acc.addEntry(subRowEntry{sr: sr, cmdIndex: cmdIndex})
			}
		}
		return
	}

	groupsPerWay := wc / groupSize
	threshold := bp.config.RepeatCntOffset * groupsPerWay

	if repeatCnt <= threshold {
		for r := 0; r < repeatCnt; r++ {
			expanded := expandSubRowGroup(*normalSR, model.SubRowStatusNormal)
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
			expanded := expandSubRowGroup(*normalSR, model.SubRowStatusNormal)
			for _, sr := range expanded {
				acc.addEntry(subRowEntry{sr: sr, cmdIndex: cmdIndex})
			}
		}
	}
}

// processRepeatMacrosCommand 情況 1A：
// DES 有 Macros 陣列且 RepeatCnt > 1
// 展開一組 Macros 產生 SubRow，flush 後最後一個 PrintNode header 改為 JNI "."
// func (s *SingleWayStrategy) processRepeatMacrosCommand(bp *basePrinter, acc *accumulator, normalSR *model.SubRow, cmdIndex int) {
// 	expanded := expandSubRowGroup(*normalSR, model.SubRowStatusNormal)
// 	for _, sr := range expanded {
// 		acc.addEntry(subRowEntry{sr: sr, cmdIndex: cmdIndex})
// 	}

// 	acc.flush()

// 	if len(acc.result) > 0 {
// 		lastPN := acc.result[len(acc.result)-1]
// 		lastPN.Header.Operator = bp.config.JNIKeyword
// 		lastPN.Header.OperatorValue = fmt.Sprintf(".-%d", 1)
// 		lastPN.Header.Interrupt = false
// 		lastPN.SourceLineNum = normalSR.SourceLineNum
// 	}
// }

// processRepeatMacrosCommand 情況 1A：
// DES 有 Macros 陣列且 RepeatCnt > 1
// 不展開一組 Macros 產生 SubRow, 只抓取D_, 或DX_, 可以不用形成波型，flush 後最後一個 PrintNode header 改為 IDXI
func (s *SingleWayStrategy) processRepeatMacrosCommand(bp *basePrinter, acc *accumulator, normalSR *model.SubRow, cmdIndex int) {
	expanded := expandSubRowGroup(*normalSR, model.SubRowStatusNormal)
	// for _, sr := range expanded {
	// 	acc.addEntry(subRowEntry{sr: sr, cmdIndex: cmdIndex})
	// }

	acc.addEntry(subRowEntry{sr: expanded[0], cmdIndex: cmdIndex})
	acc.flush()

	repeatCnt := util.ToHex[int64](int64(normalSR.RepeatCnt-2), "#", true)
	if len(acc.result) > 0 {
		lastPN := acc.result[len(acc.result)-1]
		lastPN.Header.Operator = "IDXI5"
		lastPN.Header.OperatorValue = fmt.Sprintf("%v", repeatCnt)
		lastPN.Header.Interrupt = false
		lastPN.SourceLineNum = normalSR.SourceLineNum
	}
}

func (s *SingleWayStrategy) ProcessLoop(nodes []*core.SeqNode, bp *basePrinter, acc *accumulator, i int, mergedLabel string) int {
	node := nodes[i]
	acc.flush()

	ctx := NewLoopContext(bp, acc, node, mergedLabel)
	s.loopStrategy.ProcessLoop(ctx)

	return i + 1
}
