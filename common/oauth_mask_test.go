package common

import (
	"strings"
	"testing"
)

func TestMaskSensitiveInfoMasksOAuthTokens(t *testing.T) {
	in := `Authorization: Bearer sk-ant-oat01-abc123DEF ` +
		`{"access_token":"sk-ant-acc-xyz","refresh_token":"sk-ant-ref-999","id_token":"jwt.aaa.bbb"}`
	out := MaskSensitiveInfo(in)

	for _, secret := range []string{
		"sk-ant-oat01-abc123DEF", // Bearer
		"sk-ant-acc-xyz",         // access_token
		"sk-ant-ref-999",         // refresh_token
		"jwt.aaa.bbb",            // id_token
	} {
		if strings.Contains(out, secret) {
			t.Fatalf("secret %q not masked in: %s", secret, out)
		}
	}
	if !strings.Contains(out, "***") {
		t.Fatalf("expected redaction marker in output: %s", out)
	}
}
