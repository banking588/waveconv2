package printer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"common/log"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/core"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/model/mrstatusreg"
	lpddr5mr "waveconv/pkg/patconverter/timingcheck/ddr5checker/model/mrstatusreg/lpddr5"
	"waveconv/pkg/patconverter/timingcheck/ddr5checker/regcontroller"
	t5833lpddr5 "waveconv/pkg/patconverter/timingcheck/ddr5checker/regcontroller/t5833/lpddr5"
)

// --- Command conversion tests ---

func GET_DDR5_SEQ_FILE_CP() string {
	var fpath = filepath.Join("..", "..", "testcase", "lpddr5")
	//var testFile = filepath.Join(fpath, "small_test.seq")
	//var testFile = filepath.Join(fpath, "249_ehrdcinh.seq")
	var testFile = filepath.Join(fpath, "260_ehrdcinh.seq")
	//var testFile = filepath.Join(fpath, "283_ejdbilp.seq")
	return testFile
}

func TestT5833_File(t *testing.T) {
	log.ApplyDevelopLogger(nil)
	inputBytes, err := os.ReadFile(GET_DDR5_SEQ_FILE_CP())
	if err != nil {
		t.Fatalf("讀取測試輸入檔失敗: %v", err)
	}

	//解析成SeqNode
	input := string(inputBytes)
	nodes, err := core.ParseSeqFile(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseInput: %v", err)
	}

	//處理SeqNode, 把不必要的節點刪除
	vars := core.NewVariableStore()

	var mrs mrstatusreg.MRS
	mrs = lpddr5mr.NewLPDDR5MRS()
	newNodes, err := core.ParseSeqNodes(nodes, vars, mrs, model.LPDDR5)

	//把SeqNode轉換成PrintNode, 含macro的mapper
	p, _ := NewWithWay(model.TesterT5833, 1, model.ProtocolLPDDR5)
	p.SetMRS(mrs)
	pnodes := p.Convert(newNodes)

	// Reg运算(只需要 PrintNode，不需要 SeqNode)
	// 根據需求實作CommandWriter(ex, macrodef, lp的WCK, ddr的DQS)
	op := regcontroller.NewRegOperator(
		regcontroller.WithCommandWriter(t5833lpddr5.T5833CommandWriter{
			Mrs: mrs,
		}),
	)
	op.Apply(pnodes)
	output := p.Format(pnodes)

	t.Logf("HS5503 WR 16-byte DQS spread output:\n%s", output)
}
