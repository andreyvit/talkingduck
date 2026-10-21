package telemetry

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func pbInt(field, value uint64) []byte {
	return binary.AppendUvarint(binary.AppendUvarint(nil, field<<3), value)
}
func pbMsg(field uint64, data []byte) []byte {
	b := binary.AppendUvarint(nil, field<<3|2)
	b = binary.AppendUvarint(b, uint64(len(data)))
	return append(b, data...)
}
func grpcFrame(flag byte, data []byte) []byte {
	b := []byte{flag, 0, 0, 0, 0}
	binary.BigEndian.PutUint32(b[1:], uint32(len(data)))
	return append(b, data...)
}
func grokFixture(start time.Time, days int, used *float32, extra []byte) []byte {
	period := pbInt(1, 2)
	period = append(period, pbMsg(2, pbInt(1, uint64(start.Unix())))...)
	period = append(period, pbMsg(3, pbInt(1, uint64(start.Add(time.Duration(days)*24*time.Hour).Unix())))...)
	config := pbMsg(8, period)
	if used != nil {
		config = append(config, 13)
		config = binary.LittleEndian.AppendUint32(config, math.Float32bits(*used))
	}
	config = append(config, extra...)
	return append(grpcFrame(0, pbMsg(1, config)), grpcFrame(128, []byte("grpc-status: 0\r\n"))...)
}
func TestGrokBillingExplicitAndImplicitZero(t *testing.T) {
	now := time.Date(2026, 10, 21, 7, 0, 0, 0, time.UTC)
	start := now.Add(-24 * time.Hour)
	used := float32(29)
	for _, v := range []*float32{nil, &used} {
		r, reset, err := parseGrokBilling(grokFixture(start, 7, v, pbMsg(99, []byte{0xff})), now)
		want := 100.0
		if v != nil {
			want = 71
		}
		if err != nil || r != want || !reset.Equal(start.Add(7*24*time.Hour)) {
			t.Fatalf("%v %v %v", r, reset, err)
		}
	}
	badFloat := float32(math.NaN())
	duplicate := append([]byte{13}, make([]byte, 4)...)
	valid := grokFixture(start, 7, nil, nil)
	for _, bad := range [][]byte{
		nil, valid[:len(valid)-1], grokFixture(start, 30, nil, nil), grokFixture(now.Add(time.Hour), 7, nil, nil), grokFixture(now.Add(-8*24*time.Hour), 7, nil, nil),
		grokFixture(start, 7, &badFloat, nil), grokFixture(start, 7, &used, duplicate), grokFixture(start, 7, nil, []byte{0}), grokFixture(start, 7, nil, []byte{0x80}),
		append(grpcFrame(0, pbMsg(1, nil)), grpcFrame(128, []byte("grpc-status: 0\r\n"))...),
		append(grpcFrame(0, pbMsg(1, nil)), grpcFrame(128, []byte("grpc-status: 16\r\n"))...),
	} {
		if _, _, err := parseGrokBilling(bad, now); err == nil {
			t.Fatalf("accepted invalid %x", bad)
		}
	}
}
func TestGrokFallbackRequestAndProxyReset(t *testing.T) {
	now := time.Date(2026, 10, 21, 7, 0, 0, 0, time.UTC)
	start := now.Add(-24 * time.Hour)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.Method != "POST" || r.Header.Get("Authorization") != "Bearer synthetic" || r.Header.Get("x-grpc-web") != "1" || string(body) != string([]byte{0, 0, 0, 0, 2, 8, 0}) {
			t.Error("invalid billing request")
		}
		w.Header().Set("Content-Type", "application/grpc-web+proto")
		w.Write(grokFixture(start, 7, nil, nil))
	}))
	defer server.Close()
	exact := start.Add(7*24*time.Hour + 123*time.Millisecond)
	proxy := []byte(fmt.Sprintf(`{"config":{"currentPeriod":{"end":%q}}}`, exact.Format(time.RFC3339Nano)))
	value, reset, err := grokFallback(context.Background(), server.Client(), server.URL, "synthetic", proxy, now)
	if err != nil || value != 100 || !reset.Equal(exact) {
		t.Fatalf("%v %v %v", value, reset, err)
	}
	proxy = []byte(fmt.Sprintf(`{"config":{"currentPeriod":{"end":%q}}}`, exact.Add(time.Hour).Format(time.RFC3339Nano)))
	if _, _, err := grokFallback(context.Background(), server.Client(), server.URL, "synthetic", proxy, now); err == nil {
		t.Fatal("mixed periods")
	}
}
