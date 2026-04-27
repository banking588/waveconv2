package printer

import (
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/core"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model"
)

// LoopStrategy 定義 Loop/JNI/IDXI 的封裝策略
type LoopStrategy interface {
	ProcessLoop(ctx *LoopContext) int
	ResolveJNINode(ctx *LoopContext, label string)
	FlushAsIDXI(ctx *LoopContext, srs []model.SubRow, repeatCount int)
	IsLoopOperator(op string) bool
}

// LoopContext 把 Loop 處理需要的上下文集中在一起
type LoopContext struct {
	BP          *basePrinter
	Acc         *accumulator
	Node        *core.SeqNode
	MergedLabel string
}

func NewLoopContext(bp *basePrinter, acc *accumulator, node *core.SeqNode, mergedLabel string) *LoopContext {
	return &LoopContext{
		BP:          bp,
		Acc:         acc,
		Node:        node,
		MergedLabel: mergedLabel,
	}
}
