package utils

import "testing"

func TestPluginPanicHandler(t *testing.T) {
	_, err := PluginPanicHandler(ConditionFunc(func() (string, error) {
		panic("Test panic error")
	}))
	if err == nil {
		t.Error("Expected err not be nil, but got nil")
	}
}
