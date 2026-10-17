package main

import "testing"

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    options
		wantErr bool
	}{
		{"defaults to hunks with context 3", []string{"L", "R"}, options{leftDir: "L", rightDir: "R", context: 3}, false},
		{"full", []string{"--full", "L", "R"}, options{leftDir: "L", rightDir: "R", full: true, context: 3}, false},
		{"flags after operands", []string{"L", "R", "--full"}, options{leftDir: "L", rightDir: "R", full: true, context: 3}, false},
		{"explicit hunks overrides full", []string{"--full", "--hunks", "L", "R"}, options{leftDir: "L", rightDir: "R", context: 3}, false},
		{"context space separated", []string{"--context", "5", "L", "R"}, options{leftDir: "L", rightDir: "R", context: 5}, false},
		{"context equals", []string{"--context=7", "L", "R"}, options{leftDir: "L", rightDir: "R", context: 7}, false},
		{"editor keyword", []string{"--editor", "zed", "L", "R"}, options{leftDir: "L", rightDir: "R", editor: "zed", context: 3}, false},
		{"editor equals with command", []string{"--editor=subl -n -w", "L", "R"}, options{leftDir: "L", rightDir: "R", editor: "subl -n -w", context: 3}, false},
		{"editor missing value", []string{"--editor"}, options{}, true},
		{"missing operands", []string{"--full"}, options{}, true},
		{"too many operands", []string{"L", "R", "X"}, options{}, true},
		{"unknown flag", []string{"--wat", "L", "R"}, options{}, true},
		{"context missing value", []string{"--context"}, options{}, true},
		{"context not a number", []string{"--context", "x", "L", "R"}, options{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
