package main

import (
	"testing"
)

func TestGetPSO2CoatOfArms(t *testing.T) {
	actual, err := GetPSO2CoatOfArms()
	if err != nil {
		t.Fatal(err)
	}
	t.Errorf("got %v\nwant", actual)
}
