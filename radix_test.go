package radix

import (
	"bytes"
	crand "crypto/rand"
	"fmt"
	"reflect"
	"testing"
)

func TestRadix(t *testing.T) {
	var min, max []byte
	inp := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		gen := generateIP()
		inp[string(gen)] = i
		if bytes.Compare(gen, min) < 0 || i == 0 {
			min = gen
		}
		if bytes.Compare(gen, max) > 0 || i == 0 {
			max = gen
		}
	}

	r := NewFromMap(inp)
	if r.Len() != len(inp) {
		t.Fatalf("bad length: %v %v", r.Len(), len(inp))
	}

	r.Walk(func(k []byte, v interface{}) bool {
		println(k)
		return false
	})

	for k, v := range inp {
		out, ok := r.Get([]byte(k))
		if !ok {
			t.Fatalf("missing key: %v", k)
		}
		if out != v {
			t.Fatalf("value mis-match: %v %v", out, v)
		}
	}

	// Check min and max
	outMin, _, _ := r.Minimum()
	if !bytes.Equal(outMin, min) {
		t.Fatalf("bad minimum: %v %v", outMin, min)
	}
	outMax, _, _ := r.Maximum()
	if !bytes.Equal(outMax, max) {
		t.Fatalf("bad maximum: %v %v", outMax, max)
	}

	for k, v := range inp {
		out, ok := r.Delete([]byte(k))
		if !ok {
			t.Fatalf("missing key: %v", k)
		}
		if out != v {
			t.Fatalf("value mis-match: %v %v", out, v)
		}
	}
	if r.Len() != 0 {
		t.Fatalf("bad length: %v", r.Len())
	}
}

func TestRoot(t *testing.T) {
	r := New()
	_, ok := r.Delete([]byte{})
	if ok {
		t.Fatalf("bad")
	}
	_, ok = r.Insert([]byte{}, true)
	if ok {
		t.Fatalf("bad")
	}
	val, ok := r.Get([]byte{})
	if !ok || val != true {
		t.Fatalf("bad: %v", val)
	}
	val, ok = r.Delete([]byte{})
	if !ok || val != true {
		t.Fatalf("bad: %v", val)
	}
}

func TestDelete(t *testing.T) {

	r := New()

	s := [][]byte{[]byte{}, []byte{2}, []byte{2, 5}}

	for _, ss := range s {
		r.Insert(ss, true)
	}

	for _, ss := range s {
		_, ok := r.Delete(ss)
		if !ok {
			t.Fatalf("bad %q", ss)
		}
	}
}

func TestDeletePrefix(t *testing.T) {
	type exp struct {
		inp        [][]byte
		prefix     []byte
		out        [][]byte
		numDeleted int
	}

	cases := []exp{
		{[][]byte{[]byte{}, []byte{2}, []byte{2, 3}, []byte{2, 3, 4}, []byte{33}, []byte{44}}, []byte{2}, [][]byte{[]byte{}, []byte{33}, []byte{44}}, 3},
		{[][]byte{[]byte{}, []byte{2}, []byte{2, 3}, []byte{2, 3, 4}, []byte{33}, []byte{44}}, []byte{2, 3, 4}, [][]byte{[]byte{}, []byte{2}, []byte{2, 3}, []byte{33}, []byte{44}}, 1},
		{[][]byte{[]byte{}, []byte{2}, []byte{2, 3}, []byte{2, 3, 4}, []byte{33}, []byte{44}}, []byte{}, [][]byte{}, 6},
		{[][]byte{[]byte{}, []byte{2}, []byte{2, 3}, []byte{2, 3, 4}, []byte{33}, []byte{44}}, []byte{44}, [][]byte{[]byte{}, []byte{2}, []byte{2, 3}, []byte{2, 3, 4}, []byte{33}}, 1},
		{[][]byte{[]byte{}, []byte{2}, []byte{2, 3}, []byte{2, 3, 4}, []byte{33}, []byte{44}}, []byte{45}, [][]byte{[]byte{}, []byte{2}, []byte{2, 3}, []byte{2, 3, 4}, []byte{33}, []byte{44}}, 0},
	}

	for _, test := range cases {
		r := New()
		for _, ss := range test.inp {
			r.Insert(ss, true)
		}

		deleted := r.DeletePrefix(test.prefix)
		if deleted != test.numDeleted {
			t.Fatalf("Bad delete, expected %v to be deleted but got %v", test.numDeleted, deleted)
		}

		out := [][]byte{}
		fn := func(s []byte, v interface{}) bool {
			out = append(out, s)
			return false
		}
		r.Walk(fn)

		if !reflect.DeepEqual(out, test.out) {
			t.Fatalf("mis-match: %v %v", out, test.out)
		}
	}
}

func TestLongestPrefix(t *testing.T) {
	r := New()

	keys := [][]byte{
		[]byte{},
		[]byte{0x20},
		[]byte{0x20, 0x01},
		[]byte{0x20, 0x01, 0x0d},
		[]byte{0x20, 0x01, 0x0d, 0xb8},
		[]byte{0x20, 0x02},
	}
	for _, k := range keys {
		r.Insert(k, nil)
	}
	if r.Len() != len(keys) {
		t.Fatalf("bad len: %v %v", r.Len(), len(keys))
	}

	type exp struct {
		inp []byte
		out []byte
	}
	cases := []exp{
		{[]byte{0x2}, []byte{}},
		{[]byte{0x2, 0x3, 0x4}, []byte{}},
		{[]byte{0x21}, []byte{}},
		{[]byte{0x20}, []byte{0x20}},
		{[]byte{0x20, 0x00}, []byte{0x20}},
		{[]byte{0x20, 0x01}, []byte{0x20, 0x01}},
		{[]byte{0x20, 0x01, 0xdd}, []byte{0x20, 0x01}},
		{[]byte{0x20, 0x01, 0x0d}, []byte{0x20, 0x01, 0x0d}},
		{[]byte{0x20, 0x01, 0x0d, 0xbf}, []byte{0x20, 0x01, 0x0d}},
		{[]byte{0x20, 0x01, 0x0d, 0xb8}, []byte{0x20, 0x01, 0x0d, 0xb8}},
		{[]byte{0x20, 0x02}, []byte{0x20, 0x02}},
		{[]byte{0x20, 0x02, 0x05, 0xff}, []byte{0x20, 0x02}},
	}

	for _, test := range cases {
		m, _, ok := r.LongestPrefix(test.inp)
		if !ok {
			t.Fatalf("no match: %v", test)
		}
		if !bytes.Equal(m, test.out) {
			t.Fatalf("mis-match: %v %v", m, test)
		}
	}
}

func TestWalkPrefix(t *testing.T) {
	r := New()

	keys := [][]byte{
		[]byte{},
		[]byte{0x20},
		[]byte{0x20, 0x01},
		[]byte{0x20, 0x01, 0x0d},
		[]byte{0x20, 0x01, 0x0d, 0xb8},
		[]byte{0x20, 0x02},
	}
	for _, k := range keys {
		r.Insert(k, nil)
	}
	if r.Len() != len(keys) {
		t.Fatalf("bad len: %v %v", r.Len(), len(keys))
	}

	type exp struct {
		inp []byte
		out []byte
	}
	cases := []exp{
		{
			[]byte{0x1},
			[]byte{},
		},
		{
			[]byte{0x20},
			[]byte{0x20, 0x20, 0x01, 0x20, 0x01, 0x0d, 0x20, 0x01, 0x0d, 0xb8, 0x20, 0x02},
		},
		{
			[]byte{0x20, 0x20},
			[]byte{},
		},
		{
			[]byte{0x20, 0x01},
			[]byte{0x20, 0x01, 0x20, 0x01, 0x0d, 0x20, 0x01, 0x0d, 0xb8},
		},
		{
			[]byte{0x20, 0x01, 0x0d},
			[]byte{0x20, 0x01, 0x0d, 0x20, 0x01, 0x0d, 0xb8},
		},
		{
			[]byte{0x20, 0x01, 0x0d, 0xff},
			[]byte{},
		},
		{
			[]byte{0x20, 0x01, 0x0d, 0xb8},
			[]byte{0x20, 0x01, 0x0d, 0xb8},
		},
		{
			[]byte{0x20, 0x01, 0x0d, 0xb8, 0xff},
			[]byte{},
		},
		{
			[]byte{0xb8},
			[]byte{},
		},
	}

	for _, test := range cases {
		out := []byte{}
		fn := func(s []byte, v interface{}) bool {
			out = append(out, s...)
			return false
		}
		r.WalkPrefix(test.inp, fn)
		if !reflect.DeepEqual(out, test.out) {
			t.Fatalf("mis-match: %v %v", out, test.out)
		}
	}
}

func TestWalkPath(t *testing.T) {
	r := New()

	keys := [][]byte{
		[]byte{},
		[]byte{0x20},
		[]byte{0x20, 0x01},
		[]byte{0x20, 0x01, 0x0d},
		[]byte{0x20, 0x01, 0x0d, 0xb8},
		[]byte{0x20, 0x02},
	}
	for _, k := range keys {
		r.Insert(k, nil)
	}
	if r.Len() != len(keys) {
		t.Fatalf("bad len: %v %v", r.Len(), len(keys))
	}

	type exp struct {
		inp []byte
		out []byte
	}
	cases := []exp{
		{
			[]byte{0x1},
			[]byte{},
		},
		{
			[]byte{0x20},
			[]byte{0x20},
		},
		{
			[]byte{0x20, 0x20},
			[]byte{0x20},
		},
		{
			[]byte{0x20, 0x01},
			[]byte{0x20, 0x20, 0x01},
		},
		{
			[]byte{0x20, 0x01, 0x0d},
			[]byte{0x20, 0x20, 0x01, 0x20, 0x01, 0x0d},
		},
		{
			[]byte{0x20, 0x01, 0x0d, 0xff},
			[]byte{0x20, 0x20, 0x01, 0x20, 0x01, 0x0d},
		},
		{
			[]byte{0x20, 0x01, 0x0d, 0xb8},
			[]byte{0x20, 0x20, 0x01, 0x20, 0x01, 0x0d, 0x20, 0x01, 0x0d, 0xb8},
		},
		{
			[]byte{0x20, 0x01, 0x0d, 0xb8, 0xff},
			[]byte{0x20, 0x20, 0x01, 0x20, 0x01, 0x0d, 0x20, 0x01, 0x0d, 0xb8},
		},
		{
			[]byte{0xb8},
			[]byte{},
		},
	}

	for _, test := range cases {
		out := []byte{}
		fn := func(s []byte, v interface{}) bool {
			out = append(out, s...)
			return false
		}
		r.WalkPath(test.inp, fn)
		if !reflect.DeepEqual(out, test.out) {
			t.Fatalf("mis-match: %v %v", out, test.out)
		}
	}
}

func BenchmarkInsertIPv4(b *testing.B) {
	buf := make([]byte, 16)

	radix := New()

	for i := 0; i < b.N; i++ {
		if _, err := crand.Read(buf); err != nil {
			panic(fmt.Errorf("failed to read random bytes: %v", err))
		}
		radix.Insert(buf, 64)
	}
}
func BenchmarkInsertIPv6(b *testing.B) {
	radix := New()

	for i := 0; i < b.N; i++ {
		gen := generateIP()
		radix.Insert(gen, 64)
	}
}

func BenchmarkLookupIPv4(b *testing.B) {
	buf := make([]byte, 4)
	radix := New()

	for i := 0; i < 1000000; i++ {
		if _, err := crand.Read(buf); err != nil {
			panic(fmt.Errorf("failed to read random bytes: %v", err))
		}
		radix.Insert(buf, 64)
	}
	for i := 0; i < b.N; i++ {
		if _, err := crand.Read(buf); err != nil {
			panic(fmt.Errorf("failed to read random bytes: %v", err))
		}
		radix.Get(buf)
	}
}
func BenchmarkLookupIPv6(b *testing.B) {
	buf := make([]byte, 16)
	radix := New()

	for i := 0; i < 1000000; i++ {
		if _, err := crand.Read(buf); err != nil {
			panic(fmt.Errorf("failed to read random bytes: %v", err))
		}
		radix.Insert(buf, 64)
	}

	for i := 0; i < b.N; i++ {
		if _, err := crand.Read(buf); err != nil {
			panic(fmt.Errorf("failed to read random bytes: %v", err))
		}
		radix.Get(buf)
	}
}

// generateIP is used to generate a random IP
func generateIP() []byte {
	buf := make([]byte, 16)
	if _, err := crand.Read(buf); err != nil {
		panic(fmt.Errorf("failed to read random bytes: %v", err))
	}
	buf[0] = 0x20
	buf[1] = 0x01
	buf[2] = 0x0d
	buf[3] = 0xb8

	return buf
}
