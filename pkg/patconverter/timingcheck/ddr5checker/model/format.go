package model

import (
	"fmt"
	"strings"

	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model/mrstatusreg"
)

type Formatter interface {
	Name() string
	Format(nodes []*PrintNode) string
	GetVarMapper() VarMapper
	GetConfig() *Config
	SetMRS(mrs mrstatusreg.MRS)
}

// --- 型別定義 ---
type VarMapper interface {
	SpecialtagToAddressDefine(name string) string
}
type SubRowCommandMapper func(cmd *Command, wayCount int) []SubRow

type ExpressionConverter func(exprString string) string

type PadSubRowFunc func() SubRow

// --- 配置 ---

type Config struct {
	RowMapCmd       SubRowCommandMapper
	ExpressionConv  ExpressionConverter // 處理寄存的轉換
	LabelPrefix     string
	JNIKeyword      string
	PadSubRow       PadSubRowFunc
	WayCount        int
	RepeatCntOffset int             //用來標記隐性的次數, 如advantest次數需要固定-2次, 寫2即可
	MRS             mrstatusreg.MRS // 外部注入
}

// --- 格式化配置 ---

type FormatConfig struct {
	LabelWidth    int
	OperatorWidth int
	AddrWidth     int
	SettingWidth  int
	DataWidth     int
	MacroCmdWidth int
	OptionsWidth  int
}

func DefaultFormatConfig() *FormatConfig {
	return &FormatConfig{
		LabelWidth:    20,
		OperatorWidth: 8,
		AddrWidth:     12,
		SettingWidth:  8,
		DataWidth:     12,
		MacroCmdWidth: 12,
		OptionsWidth:  20,
	}
}

// --- Violation 格式化 ---

func FormatViolation(v *TimingViolation) string {
	return fmt.Sprintf("%s: %s (expected>=%d, actual=%d)",
		v.CheckerName, v.Rule, v.Expected, v.Actual)
}

func FormatViolationMark(violations []*TimingViolation) string {
	if len(violations) == 0 {
		return ""
	}
	var parts []string
	for _, v := range violations {
		parts = append(parts, FormatViolation(v))
	}
	return fmt.Sprintf("◀◀ VIOLATION: %s", strings.Join(parts, " | "))
}

// TODO: 目前移到各個類型的formater裡了, 這裡可以刪掉也可以保留
// 因為每個機型或產品會有不同的mapping方式
// func MakeExprConverter(mapper VarMapper) ExpressionConverter {
// 	return func(strExpr string) string {
// 		fields, err := util.ParseSpecialTagAssignment(strExpr)
// 		if err != nil {
// 			log.Error(err)
// 			return ""
// 		}

// 		varName := fields[0]
// 		op := fields[1]
// 		val := fields[2]
// 		mapped := mapper.SpecialtagToAddressDefine(varName)

// 		switch op {
// 		case "=":
// 			return fmt.Sprintf("%s<%s", mapped, val)
// 		case "+=":
// 			return fmt.Sprintf("%s<%s+%s", mapped, mapped, val)
// 		case "-=":
// 			return fmt.Sprintf("%s<%s-%s", mapped, mapped, val)
// 		default:
// 			return fmt.Sprintf("%s %s %s", mapped, op, val)
// 		}
// 	}
// }
