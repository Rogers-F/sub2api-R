package service

import (
	"regexp"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestMarshalPythonStyleJSONObject(t *testing.T) {
	t.Parallel()

	reflectPayload := marshalPythonStyleJSONObject(
		paygPythonRawField("user_id", "1"),
		paygPythonStringField("amount_yuan", "1"),
	)
	got := marshalPythonStyleJSONObject(
		paygPythonStringField("terminal_sn", "100108880053500132"),
		paygPythonStringField("client_sn", "XS1743321600000abc123"),
		paygPythonStringField("total_amount", "100"),
		paygPythonStringField("payway", "1"),
		paygPythonStringField("subject", "星算code充值 ¥1"),
		paygPythonStringField("operator", "system"),
		paygPythonStringField("reflect", reflectPayload),
	)

	require.Equal(
		t,
		`{"terminal_sn": "100108880053500132", "client_sn": "XS1743321600000abc123", "total_amount": "100", "payway": "1", "subject": "\u661f\u7b97code\u5145\u503c \u00a51", "operator": "system", "reflect": "{\"user_id\": 1, \"amount_yuan\": \"1\"}"}`,
		got,
	)
}

func TestGeneratePaygClientSN(t *testing.T) {
	t.Parallel()

	got, err := generatePaygClientSN()
	require.NoError(t, err)
	require.Regexp(t, regexp.MustCompile(`^XS\d{13}[0-9a-f]{6}$`), got)
}

func TestBuildPaygSubject(t *testing.T) {
	t.Parallel()

	require.Equal(t, "星算code充值 ¥1", buildPaygSubject(" 星算code ", "1"))
	require.Equal(t, "Sub2API充值 ¥8.88", buildPaygSubject("", "8.88"))
}

func TestNewPaygProviderRejectedError(t *testing.T) {
	t.Parallel()

	err := newPaygProviderRejectedError(
		&shouqianbaRequestMeta{HTTPStatus: 200},
		"400",
		"PRECREATE_FAIL",
		"invalid sign",
		"",
	)
	status := infraerrors.FromError(err)

	require.Equal(t, int32(503), status.Code)
	require.Equal(t, "PAYG_PROVIDER_REJECTED", status.Reason)
	require.Equal(t, "200", status.Metadata["provider_http_status"])
	require.Equal(t, "400", status.Metadata["provider_result_code"])
	require.Equal(t, "PRECREATE_FAIL", status.Metadata["provider_biz_result_code"])
	require.Equal(t, "invalid sign", status.Metadata["provider_error"])
}
