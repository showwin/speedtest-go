package speedtest

import (
	"testing"
)

func BenchmarkFmt(b *testing.B) {
	bt := ByteRate(1002031.0)
	for i := 0; i < b.N; i++ {
		_ = bt.Byte(UnitTypeDecimalBits)
	}
}

func BenchmarkDefaultFmt(b *testing.B) {
	bt := ByteRate(1002031.0)
	for i := 0; i < b.N; i++ {
		_ = bt.String()
	}
}

func TestFmt(t *testing.T) {
	testData := []struct {
		rate   ByteRate
		format string
		t      UnitType
	}{
		{123123123.123, "984.98 Mbps", UnitTypeDecimalBits},
		{1231231231.123, "9.85 Gbps", UnitTypeDecimalBits},
		{123123.123, "984.98 Kbps", UnitTypeDecimalBits},
		{123.1, "984.80 bps", UnitTypeDecimalBits},

		{123123123.123, "123.12 MB/s", UnitTypeDecimalBytes},
		{1231231231.123, "1.23 GB/s", UnitTypeDecimalBytes},
		{123123.123, "123.12 KB/s", UnitTypeDecimalBytes},
		{123.1, "123.10 B/s", UnitTypeDecimalBytes},

		{123123123.123, "939.35 KiMbps", UnitTypeBinaryBits},
		{1231231231.123, "9.17 KiGbps", UnitTypeBinaryBits},
		{123123.123, "961.90 Kibps", UnitTypeBinaryBits},
		{123.1, "0.96 Kibps", UnitTypeBinaryBits},

		{123123123.123, "117.42 MiB/s", UnitTypeBinaryBytes},
		{1231231231.123, "1.15 GiB/s", UnitTypeBinaryBytes},
		{123123.123, "120.24 KiB/s", UnitTypeBinaryBytes},
		{123.1, "0.12 KiB/s", UnitTypeBinaryBytes},

		{-1, "N/A", UnitTypeBinaryBytes},
		{0, "0.00 Mbps", UnitTypeDecimalBits},
	}

	if testData[0].rate.String() != testData[0].format {
		t.Errorf("got: %s, want: %s", testData[0].rate.String(), testData[0].format)
	}

	for _, v := range testData {
		if got := v.rate.Byte(v.t); got != v.format {
			t.Errorf("got: %s, want: %s", got, v.format)
		}
	}
}
