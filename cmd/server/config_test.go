package main

import "testing"

func TestConfigDefaultsAndPort(t *testing.T) {
	config, err := ParseConfig(nil, func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if config.Address != defaultAddress {
		t.Fatalf("address=%s", config.Address)
	}
	config, err = ParseConfig(nil, func(name string) string {
		if name == "PORT" {
			return "19991"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	if config.Address != "127.0.0.1:19991" {
		t.Fatalf("address=%s", config.Address)
	}
}

func TestConfigRejectsNonLoopback(t *testing.T) {
	if _, err := ParseConfig([]string{"-addr=0.0.0.0:19081"}, func(string) string { return "" }); err == nil {
		t.Fatal("expected rejection")
	}
}
