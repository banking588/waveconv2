package mrstatusreg

import "fmt"

type MR interface {
	Value() uint8
	Set(uint8)
}

type MRS interface {
	Add(regname uint32, mrvalue uint8) bool
	Get(name string) (MR, bool)
	All() map[string]MR
}

// BaseMRS 共用儲存邏輯
type BaseMRS struct {
	Mrs map[string]MR
}

func (m *BaseMRS) Get(name string) (MR, bool) {
	mr, ok := m.Mrs[name]
	return mr, ok
}

func (m *BaseMRS) All() map[string]MR {
	return m.Mrs
}

// RegName 統一格式化暫存器名稱
func RegName(regnum uint32) string {
	return fmt.Sprintf("MR%d", regnum)
}

// 泛型取得特定 MR 型別
func GetTyped[T MR](m MRS, name string) (T, error) {
	mr, ok := m.Get(name)
	if !ok {
		var zero T
		return zero, fmt.Errorf("MR %q not found", name)
	}
	typed, ok := mr.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("MR %q type mismatch", name)
	}
	return typed, nil
}
