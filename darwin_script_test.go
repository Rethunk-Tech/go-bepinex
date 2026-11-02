package bepinex

import "testing"

func TestSetScriptExecutable(t *testing.T) {
	for _, c := range []struct{ name, in, game, want string }{
		{"empty", "a=1\nexecutable_name=\"\"\nb=2\n", "Valheim", "a=1\nexecutable_name='Valheim'\nb=2\n"},
		{"spaces", "executable_name = \"x\"\n", "Stardew Valley", "executable_name = 'Stardew Valley'\n"},
		{"quote and dollar", "executable_name=\"\"\n", "It's $HOME", "executable_name='It'\\''s $HOME'\n"},
		{"absent", "echo hi\n", "G", "echo hi\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := string(SetScriptExecutable([]byte(c.in), c.game)); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}
