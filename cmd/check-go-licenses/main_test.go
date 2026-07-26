package main

import "testing"

func TestClassifyLicense(t *testing.T) {
	tests := map[string]string{
		"Apache License Version 2.0":                                                               "Apache-2.0",
		"Permission is hereby granted, free of charge, to any person":                              "MIT",
		"Permission to use, copy, modify, and distribute this software with or without fee":        "ISC",
		"Redistribution and use in source and binary forms. Neither the name of X may be used":     "BSD-3-Clause",
		"Redistribution and use in source and binary forms are permitted provided that conditions": "BSD-2-Clause",
	}
	for text, want := range tests {
		if got := classifyLicense(text); got != want {
			t.Errorf("classifyLicense(%q) = %q, want %q", text, got, want)
		}
	}
}
