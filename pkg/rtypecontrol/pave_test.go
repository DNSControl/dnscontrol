package rtypecontrol

import "testing"

func TestPaveArgs(t *testing.T) {
	tests := []struct {
		name     string
		dataArgs []any
		dataRule string
		wantErr  bool
	}{

		// Error cases
		{
			name:     "wrong number of args",
			dataArgs: []any{"test", uint16(1)},
			dataRule: "s",
			wantErr:  true,
		},
		{
			name:     "unknown arg type",
			dataArgs: []any{uint32(100)},
			dataRule: "d",
			wantErr:  true,
		},
		{
			name:     "unsupported type to uint8",
			dataArgs: []any{int32(50)},
			dataRule: "b",
			wantErr:  true,
		},
		{
			name:     "unsupported type to uint16",
			dataArgs: []any{int32(50)},
			dataRule: "w",
			wantErr:  true,
		},

		// Multiple args
		{
			name:     "multiple args mixed",
			dataArgs: []any{"test", uint8(42), uint16(1000)},
			dataRule: "sbw",
			wantErr:  false,
		},
		{
			name:     "multiple args with conversions",
			dataArgs: []any{int(100), "255", float64(50.0)},
			dataRule: "sbb",
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := PaveArgs(tt.dataArgs, tt.dataRule); (err != nil) != tt.wantErr {
				t.Errorf("PaveArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
func TestPaveArgsConversionsCompact(t *testing.T) {
	tests := []struct {
		name     string
		args     []any
		argTypes string
		want     []any
		wantErr  bool
	}{
		// String conversions - all should result in string type
		{name: "string unchanged", args: []any{"hello"}, argTypes: "s", want: []any{"hello"}},
		{name: "int to string", args: []any{int(123)}, argTypes: "s", want: []any{"123"}},
		{name: "uint8 to string", args: []any{uint8(42)}, argTypes: "s", want: []any{"42"}},
		{name: "bool to string", args: []any{true}, argTypes: "s", want: []any{"true"}},

		// uint8 ('b') conversions - all should result in uint8 type
		{name: "uint8 unchanged", args: []any{uint8(100)}, argTypes: "b", want: []any{uint8(100)}},
		{name: "uint16 to uint8 valid", args: []any{uint16(255)}, argTypes: "b", want: []any{uint8(255)}},
		{name: "int16 to uint8 valid", args: []any{int16(100)}, argTypes: "b", want: []any{uint8(100)}},
		{name: "uint to uint8 valid", args: []any{uint(200)}, argTypes: "b", want: []any{uint8(200)}},
		{name: "int to uint8 valid", args: []any{int(50)}, argTypes: "b", want: []any{uint8(50)}},
		{name: "float64 to uint8 valid", args: []any{float64(100.9)}, argTypes: "b", want: []any{uint8(100)}},
		{name: "string to uint8 valid", args: []any{"255"}, argTypes: "b", want: []any{uint8(255)}},
		{name: "string to uint8 zero", args: []any{"0"}, argTypes: "b", want: []any{uint8(0)}},

		// uint16 ('w') conversions - all should result in uint16 type
		{name: "uint16 unchanged", args: []any{uint16(1000)}, argTypes: "w", want: []any{uint16(1000)}},
		{name: "uint8 to uint16", args: []any{uint8(255)}, argTypes: "w", want: []any{uint16(255)}},
		{name: "int16 to uint16 valid", args: []any{int16(32767)}, argTypes: "w", want: []any{uint16(32767)}},
		{name: "uint to uint16 valid", args: []any{uint(65535)}, argTypes: "w", want: []any{uint16(65535)}},
		{name: "int to uint16 valid", args: []any{int(30000)}, argTypes: "w", want: []any{uint16(30000)}},
		{name: "float64 to uint16 valid", args: []any{float64(1234.5)}, argTypes: "w", want: []any{uint16(1234)}},
		{name: "string to uint16 valid", args: []any{"65535"}, argTypes: "w", want: []any{uint16(65535)}},

		// Multiple arguments
		{name: "multiple types", args: []any{uint8(1), "test", uint16(1000)}, argTypes: "bsw", want: []any{uint8(1), "test", uint16(1000)}},
		{name: "all conversions", args: []any{"100", int(200), float64(50.5)}, argTypes: "bbs", want: []any{uint8(100), uint8(200), "50.5"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PaveArgs(tt.args, tt.argTypes)
			if (err != nil) != tt.wantErr {
				t.Errorf("PaveArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				for i, want := range tt.want {
					if tt.args[i] != want {
						t.Errorf("PaveArgs() arg[%d] = %v (type %T), want %v (type %T)", i, tt.args[i], tt.args[i], want, want)
					}
				}
			}
		})
	}
}
