package core

import (
	"strconv"
	"strings"

	"common/log"

	"waveconv/pkg/util"
)

// ========================================
// VariableStore: SPECIAL_TAG 變數管理
// ========================================

type SpecialTagOperatorType int

const (
	Specified   SpecialTagOperatorType = iota // = 指定型
	Incremental                               // += 增量型
	Decremental                               // -= 減量型
)

// 增量型以及減量型的tag都return true, 即都是變化型的tag
func IsRelativeType(t SpecialTagOperatorType) bool {
	if t == Incremental || t == Decremental {
		return true
	}
	return false
}

type VariableHistory struct {
	Expression      string
	ProcessedOnLine int
}

type VariableValue struct {
	PreviousValue     int64  //记录上一次指定的值
	CurrentValue      int64  //记录当前指定的值
	CurrentExpression string //记录当前, 实际转换要打印的值
	History           []*VariableHistory
}

type VariableStore struct {
	vars map[string]*VariableValue
}

func NewVariableStore() *VariableStore {
	return &VariableStore{vars: make(map[string]*VariableValue)}
}

func (vs *VariableStore) Apply(node *SeqNode) {
	fields, err := util.ParseSpecialTagAssignment(node.ExprString)
	if err != nil {
		log.Error(err)
		return
	}

	if len(fields) > 3 && len(fields)%2 == 0 {
		//是FP7, dq setting
		log.Debugf("Processing SPECIAL_TAG: %v on line %v", node.ExprString, node.LineNum)
		node.SpecialTagDQSetting = fields
		return
	}

	log.Debugf("Processing SPECIAL_TAG: %v on line %v", node.ExprString, node.LineNum)
	lhs := fields[0] //LHS
	op := fields[1]
	rhs := fields[2] //可能是個變量, 不是常量(16進制), 要先找一次vars, RHS

	//先建立該次的name處理, 先找出vairable實例, 沒有則先建立新的
	var currentVV *VariableValue
	if existVV, ok := vs.vars[lhs]; !ok {
		currentVV = &VariableValue{
			//第一次新建的变量都预设为0, 不做比较
			CurrentValue:      0,
			PreviousValue:     0,
			CurrentExpression: "",
		}
		vs.vars[lhs] = currentVV
	} else {
		currentVV = existVV
	}

	//處理RHS
	var val int64
	if v, err := util.HexToInt64(rhs); err == nil {
		//是數字
		val = v
	} else {
		//不是常量數字, 從vars裡找, 找的到就是先前已指定過值, 直接使用, 找不到就給0
		if strings.Contains(rhs, "+") || strings.Contains(rhs, "-") || strings.Contains(rhs, "*") || strings.Contains(rhs, "/") {
			// 情況1. 又是一個算式, 直接把算式展開, ex: BG_1 + 1, BG_1 - 1
			symbols := util.ExtractVariables(rhs)

			//開始Looop Symbols, 把每個變量都在vras裡找一次
			for _, s := range symbols {
				valueStr := ""
				if exist, ok := vs.vars[s]; ok {
					valueStr = strconv.FormatInt(exist.CurrentValue, 10)
				} else {
					valueStr = "0"
				}
				rhs = strings.Replace(rhs, s, valueStr, -1)

				// TODO: 把rhs計算成值
				// val = rsh的計算
			}
		} else {
			// 情況2. 單純的變量, ex: BG_1
			if v, ok := vs.vars[rhs]; ok {
				val = v.CurrentValue
			} else {
				val = 0
			}
		}
	}

	// 記錄變量所有的變化過程
	his := &VariableHistory{
		ProcessedOnLine: node.LineNum,
		Expression:      node.ExprString,
	}
	currentVV.History = append(currentVV.History, his)
	currentVV.CurrentExpression = node.ExprString

	switch op {
	case "=":
		node.SpecialTagOpType = Specified
		currentVV.PreviousValue = currentVV.CurrentValue
		currentVV.CurrentValue = val

	case "+=":
		node.SpecialTagOpType = Incremental

		// 累加型的SPECIAL_TAG, 不要重新附值
		//vv.Current = val
		//vv.Previous = vv.Current
		//vs.vars[name] = vv

	case "-=":
		node.SpecialTagOpType = Decremental

		// 累加型的SPECIAL_TAG, 不要重新附值
		//vv.Previous = vv.Current
		//vv.Current = val
		//vs.vars[name] = vv

	}

	//TODO: node.SkipThisSpecialTagNode, 如果值一直没变过, 把SkipThisSpecialTageNode置1, SeqExpander处理时会直接跳过该Node

}

func (vs *VariableStore) Resolve(token string) int {
	if v, ok := vs.vars[token]; ok {
		return int(v.CurrentValue)
	}
	v, err := strconv.ParseInt(token, 0, 64)
	if err != nil {
		return 0
	}
	return int(v)
}

func (vs *VariableStore) ResolveHex(token string) byte {
	v, err := strconv.ParseUint(token, 0, 64)
	if err != nil {
		return 0
	}
	return byte(v & 0xFF)
}
