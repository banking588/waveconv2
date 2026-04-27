package printer

import (
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/core"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model/mrstatusreg"
)

type Printer interface {
	model.Formatter
	Convert(nodes []*core.SeqNode) []*model.PrintNode
	ConvertWithCheck(nodes []*core.SeqNode, checkResult *model.CheckResult) []*model.PrintNode
	SetWayCount(n int)
	SetLoopStrategy(s LoopStrategy)
	SetMRS(mrs mrstatusreg.MRS)
}
