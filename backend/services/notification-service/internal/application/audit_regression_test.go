package application

import (
	"fmt"
	"testing"
)

func TestRegressionSSEMustEnforceTotalConnectionLimit(t *testing.T) {
	b := NewSSEBroker()
	for n := 0; n < maxSSEClientsTotal; n++ {
		if _, err := b.Subscribe(n/maxSSEClientsPerUser, fmt.Sprint(n)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := b.Subscribe(10000, "extra"); err == nil {
		t.Fatal("global limit exceeded")
	}
	b.Unsubscribe(0, "0")
	b.Unsubscribe(0, "0")
	if _, err := b.Subscribe(10000, "extra"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Subscribe(10001, "extra"); err == nil {
		t.Fatal("double unsubscribe freed two slots")
	}
}
func TestDuplicateSSEClientDoesNotReplaceConnection(t *testing.T) {
	b := NewSSEBroker()
	first, _ := b.Subscribe(1, "same")
	if _, err := b.Subscribe(1, "same"); err == nil {
		t.Fatal("duplicate accepted")
	}
	b.SendToUser(1, "test", nil)
	select {
	case <-first.Messages:
	default:
		t.Fatal("original connection lost")
	}
}
