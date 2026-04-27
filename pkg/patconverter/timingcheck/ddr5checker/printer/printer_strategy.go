package printer

import (
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/core"
)

// PrinterStrategy 定義 wayCount ≤ 1 和 wayCount > 1 的不同處理流程
type PrinterStrategy interface {
	ProcessCommand(nodes []*core.SeqNode, bp *basePrinter, acc *accumulator, i int, cmdIndex int) (int, int)
	ProcessLoop(nodes []*core.SeqNode, bp *basePrinter, acc *accumulator, i int, mergedLabel string) int
	ShouldPad() bool
	SetLoopStrategy(ls LoopStrategy)
}
