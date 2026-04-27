package ddr5

import (
	"fmt"
	"strconv"
	"strings"

	"common/log"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model/mrstatusreg"
	"waveconv/pkg/util"
)

var nativeCmd = model.DDR5

type t5833VarMapper struct{}

func (m *t5833VarMapper) SpecialtagToAddressDefine(name string) string {
	upper := strings.ToUpper(name)
	switch {
	case strings.HasPrefix(upper, "BGBA"):
		return "Z"
	case strings.HasPrefix(upper, "ROW"):
		return "XC"
	case strings.HasPrefix(upper, "COL"):
		return "YC"
	case strings.HasPrefix(upper, "BG"):
		return "N"
	case strings.HasPrefix(upper, "BA"):
		return "B"
	default:
		return name
	}
}

func makeExprConverter(mapper model.VarMapper) model.ExpressionConverter {
	return func(strExpr string) string {
		fields, err := util.ParseSpecialTagAssignment(strExpr)
		if err != nil {
			log.Error(err)
			return ""
		}

		varName := fields[0]
		op := fields[1]
		val := fields[2]
		mapped := mapper.SpecialtagToAddressDefine(varName)

		switch op {
		case "=":
			return fmt.Sprintf("%s<%s", mapped, val)
		case "+=":
			return fmt.Sprintf("%s<%s+%s", mapped, mapped, val)
		case "-=":
			return fmt.Sprintf("%s<%s-%s", mapped, mapped, val)
		default:
			return fmt.Sprintf("%s %s %s", mapped, op, val)
		}
	}
}

// HS5503Formatter 是 5503HS 格式的 Formatter 實作。
type T5833Formatter struct {
	varMapper    *t5833VarMapper
	formatConfig *model.FormatConfig
	config       *model.Config
}

func NewT5833Formatter() *T5833Formatter {
	return NewT5833FormatterWithWay(0)
}

func NewT5833FormatterWithWay(wayCount int) *T5833Formatter {
	mapper := &t5833VarMapper{}
	f := &T5833Formatter{
		varMapper:    mapper,
		formatConfig: model.DefaultFormatConfig(),
	}
	f.config = &model.Config{
		RowMapCmd:      nil,
		MRS:            nil,
		ExpressionConv: makeExprConverter(mapper),
		LabelPrefix:    "LOOP_",
		JNIKeyword:     "JNI",
		WayCount:       wayCount,
		PadSubRow: func() model.SubRow {
			return model.SubRow{
				Macro: "D_",
			}
		},
	}
	return f
}

// SetMRS 注入外部的 MRS，同時建立依賴 MRS 的 RowMapCmd 閉包
func (f *T5833Formatter) SetMRS(mrs mrstatusreg.MRS) {
	f.config.MRS = mrs
	f.config.RowMapCmd = make5833SubRowCommandMapper(mrs)
}

func (f *T5833Formatter) Name() string { return "5503HS" }

func (f *T5833Formatter) GetVarMapper() model.VarMapper { return f.varMapper }

func (f *T5833Formatter) GetConfig() *model.Config { return f.config }

// 打印
func (f *T5833Formatter) Format(nodes []*model.PrintNode) string {
	var sb strings.Builder

	for _, node := range nodes {
		if node.Header.Label != "" {
			sb.WriteString("\n")
		}

		firstNormal := true
		for _, sr := range node.SubRows {
			if sr.Violation != nil {
				sb.WriteString(fmt.Sprintf("%-20s\t\t\t\t%s\n", "",
					model.FormatViolationMark([]*model.TimingViolation{sr.Violation})))
				continue
			}

			if firstNormal {
				if node.Header.Label != "" {
					sb.WriteString(fmt.Sprintf("%-20s:", node.Header.Label))
				} else {
					sb.WriteString(fmt.Sprintf("%-20s", ""))
				}
				sb.WriteString(fmt.Sprintf("\t%-8s", node.Header.Operator))
				if node.Header.OperatorValue != "" {
					sb.WriteString(fmt.Sprintf("%-14s", node.Header.OperatorValue))
				}
				if node.Header.Interrupt {
					sb.WriteString(fmt.Sprintf("%-4s", "I"))
				}
				firstNormal = false
			} else {
				sb.WriteString(fmt.Sprintf("%-20s", ""))
				sb.WriteString(fmt.Sprintf("\t%-8s", ""))
				if node.Header.OperatorValue != "" {
					sb.WriteString(fmt.Sprintf("%-14s", ""))
				}
			}

			//sb.WriteString(fmt.Sprintf("%-4s", ":"))
			if len(sr.Addr) > 0 {
				sb.WriteString(fmt.Sprintf("%-12s", strings.Join(sr.Addr, "\t")))
			}
			sb.WriteString(fmt.Sprintf("%-12s", sr.Macro))

			for _, s := range sr.Settings {
				sb.WriteString(fmt.Sprintf("%-8s", s))
			}

			if sr.IsPadding() {
				sb.WriteString(";; Auto Pad")
			}

			if sr.Status == model.SubRowStatusSplitByDes {
				sb.WriteString(";; SplitByDes")
			}

			if sr.SourceLineNum != 0 {
				sb.WriteString(";; from line: " + strconv.Itoa(sr.SourceLineNum))
			}

			sb.WriteString("\n")
		}
	}

	return sb.String()
}
