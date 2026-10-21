package telemetry

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

const grokBillingURL = "https://grok.com/grok_api_v2.GrokBuildBilling/GetGrokCreditsConfig"

func grokFallback(parent context.Context, client *http.Client, endpoint, token string, proxy []byte, now time.Time) (float64, *time.Time, error) {
	ctx, cancel := context.WithTimeout(parent, 6*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader([]byte{0, 0, 0, 0, 2, 8, 0}))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/grpc-web+proto")
	req.Header.Set("x-grpc-web", "1")
	req.Header.Set("Origin", "https://grok.com")
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("Grok billing fallback failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, nil, fmt.Errorf("Grok billing fallback: HTTP %d", resp.StatusCode)
	}
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/grpc-web+proto") {
		return 0, nil, fmt.Errorf("Invalid Grok billing content type")
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, (2<<20)+1))
	if err != nil || len(raw) > 2<<20 {
		return 0, nil, fmt.Errorf("Invalid Grok billing response")
	}
	value, reset, err := parseGrokBilling(raw, now)
	if err != nil {
		return 0, nil, err
	}
	// Keep the primary endpoint's exact reset, but never mix different periods.
	var x struct {
		Config struct {
			Period struct{ End json.RawMessage } `json:"currentPeriod"`
			End    json.RawMessage               `json:"billingPeriodEnd"`
		}
	}
	if json.Unmarshal(proxy, &x) == nil {
		primary := timestamp(x.Config.Period.End)
		if primary == nil {
			primary = timestamp(x.Config.End)
		}
		if primary != nil {
			if math.Abs(primary.Sub(*reset).Seconds()) >= 1 {
				return 0, nil, fmt.Errorf("Grok billing periods disagree")
			}
			reset = primary
		}
	}
	return value, reset, nil
}

type protoField struct {
	number  uint64
	wire    byte
	integer uint64
	data    []byte
}

// Decode only the wire format; opaque strings/bytes are never recursively scanned.
func billingFields(data []byte) ([]protoField, error) {
	var fields []protoField
	bad := fmt.Errorf("Malformed Grok billing protobuf")
	for len(data) > 0 {
		key, n := binary.Uvarint(data)
		if n <= 0 || key>>3 == 0 || key>>3 > 536870911 {
			return nil, bad
		}
		data = data[n:]
		f := protoField{number: key >> 3, wire: byte(key & 7)}
		switch f.wire {
		case 0:
			v, n := binary.Uvarint(data)
			if n <= 0 {
				return nil, bad
			}
			f.integer = v
			data = data[n:]
		case 1, 5:
			n := 4
			if f.wire == 1 {
				n = 8
			}
			if len(data) < n {
				return nil, bad
			}
			f.data = data[:n]
			data = data[n:]
		case 2:
			size, n := binary.Uvarint(data)
			if n <= 0 {
				return nil, bad
			}
			data = data[n:]
			if size > uint64(len(data)) {
				return nil, bad
			}
			f.data = data[:int(size)]
			data = data[int(size):]
		default:
			return nil, bad
		}
		fields = append(fields, f)
	}
	return fields, nil
}
func billingField(fields []protoField, number uint64, wire byte, required bool) (protoField, error) {
	var found protoField
	count := 0
	for _, f := range fields {
		if f.number == number {
			found = f
			count++
		}
	}
	if count > 1 || (count == 1 && found.wire != wire) || (required && count == 0) {
		return protoField{}, fmt.Errorf("Invalid Grok billing field")
	}
	return found, nil
}
func billingMessage(fields []protoField, number uint64) ([]protoField, error) {
	f, err := billingField(fields, number, 2, true)
	if err != nil {
		return nil, err
	}
	return billingFields(f.data)
}
func billingTimestamp(fields []protoField, number uint64) (time.Time, error) {
	msg, err := billingMessage(fields, number)
	if err != nil {
		return time.Time{}, err
	}
	sec, err := billingField(msg, 1, 0, true)
	if err != nil || sec.integer > math.MaxInt64 {
		return time.Time{}, fmt.Errorf("Invalid Grok reset")
	}
	nanos, err := billingField(msg, 2, 0, false)
	if err != nil || nanos.integer >= 1e9 {
		return time.Time{}, fmt.Errorf("Invalid Grok timestamp")
	}
	return time.Unix(int64(sec.integer), int64(nanos.integer)), nil
}
func parseGrokBilling(raw []byte, now time.Time) (float64, *time.Time, error) {
	bad := fmt.Errorf("Weekly Grok billing unavailable")
	var payload []byte
	statusOK := false
	for len(raw) > 0 {
		if len(raw) < 5 {
			return 0, nil, bad
		}
		flag, size := raw[0], binary.BigEndian.Uint32(raw[1:5])
		raw = raw[5:]
		if uint64(size) > uint64(len(raw)) {
			return 0, nil, bad
		}
		frame := raw[:int(size)]
		raw = raw[int(size):]
		switch flag {
		case 0:
			if payload != nil || statusOK {
				return 0, nil, bad
			}
			payload = frame
		case 128:
			if len(raw) != 0 {
				return 0, nil, bad
			}
			count := 0
			for _, line := range strings.Split(string(frame), "\r\n") {
				key, value, ok := strings.Cut(line, ":")
				if ok && strings.EqualFold(strings.TrimSpace(key), "grpc-status") {
					count++
					statusOK = strings.TrimSpace(value) == "0"
				}
			}
			if count != 1 || !statusOK {
				return 0, nil, bad
			}
		default:
			return 0, nil, bad
		}
	}
	if payload == nil || !statusOK {
		return 0, nil, bad
	}
	root, err := billingFields(payload)
	if err != nil {
		return 0, nil, err
	}
	config, err := billingMessage(root, 1)
	if err != nil {
		return 0, nil, err
	}
	period, err := billingMessage(config, 8)
	if err != nil {
		return 0, nil, err
	}
	kind, err := billingField(period, 1, 0, true)
	if err != nil || kind.integer != 2 {
		return 0, nil, bad
	}
	start, err := billingTimestamp(period, 2)
	if err != nil {
		return 0, nil, err
	}
	end, err := billingTimestamp(period, 3)
	if err != nil {
		return 0, nil, err
	}
	if start.After(now) || !end.After(now) || end.Sub(start) < 167*time.Hour || end.Sub(start) > 169*time.Hour {
		return 0, nil, bad
	}
	percent, err := billingField(config, 1, 5, false)
	if err != nil {
		return 0, nil, err
	}
	used := 0.0
	if percent.number != 0 {
		used = float64(math.Float32frombits(binary.LittleEndian.Uint32(percent.data)))
	}
	// Proto3 omits a zero scalar. Accept that default only with a complete,
	// successful response and an explicitly current weekly period.
	if math.IsNaN(used) || math.IsInf(used, 0) || used < 0 || used > 100 {
		return 0, nil, bad
	}
	return 100 - used, &end, nil
}
