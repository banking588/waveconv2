// regcontroller/hs5503.go
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

type hs5503ExprMapper struct{}

func (m *hs5503ExprMapper) SpecialtagToAddressDefine(name string) string {
	upper := strings.ToUpper(name)
	switch {
	case strings.HasPrefix(upper, "BGBA"):
		return "N"
	case strings.HasPrefix(upper, "ROW"):
		return "XC"
	case strings.HasPrefix(upper, "COL"):
		return "YC"
	case strings.HasPrefix(upper, "BG"):
		return "Z"
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
			processedVal := val
			if strings.HasPrefix(val, "0x") {
				valuePart := strings.TrimPrefix(val, "0x")
				foundNonZero := false
				for i := 0; i < len(valuePart); i++ {
					if valuePart[i] != '0' {
						processedVal = "#" + string(valuePart[i])
						foundNonZero = true
						break
					}
				}
				if !foundNonZero && len(valuePart) > 0 {
					processedVal = "#0"
				}
			}
			return fmt.Sprintf("%s<%s", mapped, processedVal)
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
type HS5503Formatter struct {
	varMapper    *hs5503ExprMapper
	formatConfig *model.FormatConfig
	config       *model.Config
}

func NewHS5503Formatter() *HS5503Formatter {
	return NewHS5503FormatterWithWay(0)
}
func NewHS5503FormatterWithWay(wayCount int) *HS5503Formatter {
	mapper := &hs5503ExprMapper{}
	f := &HS5503Formatter{
		varMapper:    mapper,
		formatConfig: model.DefaultFormatConfig(),
	}

	f.config = &model.Config{
		RowMapCmd:       nil, // SetMRS 時才建立閉包
		MRS:             nil, // 先不設
		ExpressionConv:  makeExprConverter(mapper),
		LabelPrefix:     "LOOP_",
		JNIKeyword:      "JNI",
		WayCount:        wayCount,
		RepeatCntOffset: 2,
		PadSubRow: func() model.SubRow {
			return model.SubRow{
				Macro: "D_",
			}
		},
	}
	return f
}

// SetMRS 注入外部的 MRS，同時建立依賴 MRS 的 RowMapCmd 閉包
func (f *HS5503Formatter) SetMRS(mrs mrstatusreg.MRS) {
	f.config.MRS = mrs
	f.config.RowMapCmd = makeHS5503SubRowCommandMapper(mrs)
}

func (f *HS5503Formatter) Name() string { return "5503HS" }

func (f *HS5503Formatter) GetVarMapper() model.VarMapper { return f.varMapper }

func (f *HS5503Formatter) GetConfig() *model.Config { return f.config }

func (f *HS5503Formatter) processJNIMapping(nodes []*model.PrintNode) {
	// 创建 loopCount 到 JNI 值的映射
	loopCountMap := make(map[int]string)

	for _, node := range nodes {
		if strings.HasPrefix(node.Header.Operator, "JNI") && node.Header.LoopCount > 0 {

			jniValue, exists := loopCountMap[node.Header.LoopCount]
			if !exists {
				jniIndex := len(loopCountMap) + 1
				jniValue = fmt.Sprintf("JNI%d", jniIndex)
				loopCountMap[node.Header.LoopCount] = jniValue
			}

			node.Header.Operator = jniValue
		}
	}
}

func (f *HS5503Formatter) AddExtraSubRowInfo(nodes []*model.PrintNode) {
	for _, node := range nodes {
		node.Header.Interrupt = true
	}

	// 先设置所有默认值
	for _, node := range nodes {
		for i := range node.SubRows {
			if node.SubRows[i].XYC == "" {
				node.SubRows[i].XYC = "XYC"
			}
			if node.SubRows[i].TimingGroup == "" {
				node.SubRows[i].TimingGroup = "/T1"
			}
		}
	}

	// 跨 Node 处理 XYC 推断
	for nodeIdx, node := range nodes {
		for subRowIdx := range node.SubRows {
			if len(node.SubRows[subRowIdx].Addr) > 0 {
				// 推断 XYC 值
				xyCValue := ""
				for _, addr := range node.SubRows[subRowIdx].Addr {
					rightPart := addr
					if idx := strings.Index(addr, "<"); idx != -1 {
						rightPart = addr[idx:]
					}

					switch {
					case strings.Contains(rightPart, "_2"):
						xyCValue = "XYC2"
					case strings.Contains(rightPart, "_3"):
						xyCValue = "XYC3"
					case strings.Contains(rightPart, "_4"):
						xyCValue = "XYC4"
					}

					if xyCValue != "" {
						break
					}
				}

				// 如果推断出 XYC 值，应用到后续两行（跨 Node）
				if xyCValue != "" {
					appliedCount := 0
					maxApplyCount := 2

					// 从当前行的下一行开始
					currentNodeIdx := nodeIdx
					currentSubRowIdx := subRowIdx + 1

					// 跨 Node 应用 XYC 值
					for appliedCount < maxApplyCount {
						// 检查是否还有下一个 Node
						if currentNodeIdx >= len(nodes) {
							break
						}

						// 检查是否还有下一个 SubRow
						if currentSubRowIdx >= len(nodes[currentNodeIdx].SubRows) {
							// 移动到下一个 Node
							currentNodeIdx++
							currentSubRowIdx = 0
							continue
						}

						// 只有当目标行是默认 XYC 值时才设置（避免覆盖已有设置）
						if nodes[currentNodeIdx].SubRows[currentSubRowIdx].XYC == "XYC" {
							nodes[currentNodeIdx].SubRows[currentSubRowIdx].XYC = xyCValue
						}

						appliedCount++
						currentSubRowIdx++
					}
				}
			}
		}
	}
}

// 打印
func (f *HS5503Formatter) Format(nodes []*model.PrintNode) string {
	f.processJNIMapping(nodes)
	f.AddExtraSubRowInfo(nodes)
	var sb strings.Builder

	maxLabelLength := 0
	for _, node := range nodes {
		if len(node.Header.Label) > maxLabelLength {
			maxLabelLength = len(node.Header.Label)
		}
	}
	labelWidth := maxLabelLength + 1 // 冒号占1个字符

	const (
		operatorWidth    = 12
		opValueWidth     = 16
		interruptWidth   = 4
		addrWidth        = 40
		macroWidth       = 20
		dataWidth        = 20
		timingGroupWidth = 12
		xycWidth         = 12
		settingWidth     = 12
		fieldSpacing     = 10
	)

	for _, node := range nodes {
		if node.Header.Label != "" {
			sb.WriteString("\n")
		}

		firstNormal := true
		for _, sr := range node.SubRows {
			if sr.Violation != nil {
				sb.WriteString(fmt.Sprintf("%-*s\t\t\t\t%s\n",
					labelWidth, "",
					model.FormatViolationMark([]*model.TimingViolation{sr.Violation})))
				continue
			}

			if firstNormal {
				if node.Header.Label != "" {
					sb.WriteString(fmt.Sprintf("%-*s:", labelWidth-1, node.Header.Label))
				} else {
					sb.WriteString(fmt.Sprintf("%-*s", labelWidth, ""))
				}

				sb.WriteString(strings.Repeat(" ", fieldSpacing)) //让lable后的: 和operator之间有间隔空隙

				sb.WriteString(fmt.Sprintf("%-*s", operatorWidth, node.Header.Operator))

				if node.Header.OperatorValue != "" {
					sb.WriteString(fmt.Sprintf("%-*s", opValueWidth, node.Header.OperatorValue))
				} else {
					sb.WriteString(fmt.Sprintf("%-*s", opValueWidth, ""))
				}

				if node.Header.Interrupt {
					sb.WriteString(fmt.Sprintf("%-*s", interruptWidth, "I"))
				} else {
					sb.WriteString(fmt.Sprintf("%-*s", interruptWidth, ""))
				}
				firstNormal = false
			} else {
				sb.WriteString(fmt.Sprintf("%-*s", labelWidth, ""))
				sb.WriteString(strings.Repeat(" ", fieldSpacing))
				sb.WriteString(fmt.Sprintf("%-*s", operatorWidth, ""))
				sb.WriteString(fmt.Sprintf("%-*s", opValueWidth, ""))
				sb.WriteString(fmt.Sprintf("%-*s", interruptWidth, ""))
			}

			sb.WriteString(":")
			sb.WriteString(strings.Repeat(" ", fieldSpacing))

			if len(sr.Addr) > 0 {
				addrContent := strings.Join(sr.Addr, " ")
				addrContent = strings.ReplaceAll(addrContent, "\t", "  ")
				sb.WriteString(fmt.Sprintf("%-*s", addrWidth, addrContent))
			} else {
				sb.WriteString(fmt.Sprintf("%-*s", addrWidth, ""))
			}

			if len(sr.Data) > 0 {
				dataContent := strings.Join(sr.Data, " ")
				dataContent = strings.ReplaceAll(dataContent, "\t", "  ")
				sb.WriteString(fmt.Sprintf("%-*s", dataWidth, dataContent))
			} else {
				sb.WriteString(fmt.Sprintf("%-*s", dataWidth, ""))
			}

			sb.WriteString(fmt.Sprintf("%-*s", macroWidth, sr.Macro))
			sb.WriteString(fmt.Sprintf("%-*s", xycWidth, sr.XYC))
			sb.WriteString(fmt.Sprintf("%-*s", timingGroupWidth, sr.TimingGroup))

			for _, s := range sr.Settings {
				sb.WriteString(fmt.Sprintf("%-*s", settingWidth, s))
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
