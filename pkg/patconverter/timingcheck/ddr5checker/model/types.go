package model

import (
	"fmt"
	"time"

	"innotron.com/waveconv/pkg/util"
)

type Cursor struct {
	PN int
	SR int
	XY string
}

type AddressInfo struct {
	Name  string
	Value util.HexValue
}

// Command 表示一條完整的命令
type Command struct {
	LineNum    int
	LineCount  int
	Cycle      int
	Type       CommandType
	BankGroup  *AddressInfo
	Bank       *AddressInfo
	Row        *AddressInfo
	Column     *AddressInfo
	MA         *AddressInfo
	OpCode     *AddressInfo
	CW         *AddressInfo
	DQDataTopo []string //Data
	DQDmTopo   []string //DM
	DQSetting  []string //FP7
	HasDQ      bool
	DQDelay    int
	RepeatCnt  int // DES 544 等重複拍數, 預設 1
	Raw        string
}

// BankKey 唯一標識一個 Bank
type BankKey struct {
	BankGroup util.HexValue
	Bank      util.HexValue
}

func (b BankKey) String() string {
	return fmt.Sprintf("BG%s_BA%s", b.BankGroup, b.Bank)
}

func (c *Command) GetBankKey() BankKey {
	return BankKey{BankGroup: c.BankGroup.Value, Bank: c.Bank.Value}
}

// TimingViolation 時序違規記錄
type TimingViolation struct {
	CheckerName string
	Command     *Command
	PrevCommand *Command
	Rule        string
	Expected    int
	Actual      int
	Timestamp   time.Time
}

func (v *TimingViolation) String() string {
	prevInfo := "N/A"
	if v.PrevCommand != nil {
		prevInfo = fmt.Sprintf("%s(line=%d, cycle=%d, %s)",
			v.PrevCommand.Type, v.PrevCommand.LineNum,
			v.PrevCommand.Cycle, v.PrevCommand.GetBankKey().String())
	}
	return fmt.Sprintf("[VIOLATION] %s: %s -> %s(line=%d, cycle=%d, %s), %s (expected >= %d tCK, actual = %d tCK)",
		v.CheckerName,
		prevInfo,
		v.Command.Type, v.Command.LineNum, v.Command.Cycle, v.Command.GetBankKey().String(),
		v.Rule,
		v.Expected, v.Actual,
	)
}

// CheckResult timing check 的結果
type CheckResult struct {
	Commands   []*Command
	Violations []*TimingViolation
	// ViolationsByLine 按原始行號索引 violation
	// key = LineNum, value = 該行的所有 violations
	ViolationsByLine map[int][]*TimingViolation
}

// HasViolations 是否有違規
func (r *CheckResult) HasViolations() bool {
	return len(r.Violations) > 0
}

// ViolationsForLine 取得某行的所有 violations
func (r *CheckResult) ViolationsForLine(lineNum int) []*TimingViolation {
	if r.ViolationsByLine == nil {
		return nil
	}
	return r.ViolationsByLine[lineNum]
}

// ViolationsForCycle 取得某 cycle 的所有 violations
func (r *CheckResult) ViolationsForCycle(cycle int) []*TimingViolation {
	var result []*TimingViolation
	for _, v := range r.Violations {
		if v.Command != nil && v.Command.Cycle == cycle {
			result = append(result, v)
		}
	}
	return result
}
