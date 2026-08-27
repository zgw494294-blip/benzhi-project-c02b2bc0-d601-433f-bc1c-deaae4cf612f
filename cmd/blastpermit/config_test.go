package main

import "testing"

func TestValidateAddr(t *testing.T) {
	for _, addr := range []string{"0.0.0.0:19081", "localhost:19081", "127.0.0.1:0", "127.0.0.1:70000"} {
		if validateAddr(addr) == nil {
			t.Fatalf("应拒绝地址 %s", addr)
		}
	}
	if err := validateAddr("127.0.0.1:19081"); err != nil {
		t.Fatal(err)
	}
}
