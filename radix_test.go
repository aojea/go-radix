package radix

import (
	"bytes"
	crand "crypto/rand"
	"fmt"
	"net"
	"reflect"
	"testing"
)

func TestNextIP(t *testing.T) {
	r := New()
	// Insert 1000 IPs
	for i := 0; i < 10; i++ {
		gen := generateIPv6()
		r.Insert(gen.To16(), 64)
	}
	// Check min and max
	outMin, _, _ := r.Minimum()
	fmt.Println(net.IP(outMin))
	outMax, _, _ := r.Maximum()
	fmt.Println(net.IP(outMax))
	// Check size
	fmt.Println(r.size)
	// Dump tree
	r.Walk(func(k []byte, v interface{}) bool {
		fmt.Println(net.IP(k))
		return false
	})
}

func TestRadix(t *testing.T) {
	var min, max []byte
	inp := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		gen := generateIPv6()
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
	radix := New()
	for i := 0; i < b.N; i++ {
		gen := generateIPv4()
		radix.Insert(gen.To16(), 64)
	}
}
func BenchmarkInsertIPv6(b *testing.B) {
	radix := New()
	for i := 0; i < b.N; i++ {
		gen := generateIPv6()
		radix.Insert(gen.To16(), 64)
	}
}

func BenchmarkInsertMapIPv4(b *testing.B) {
	m := make(map[string]bool)
	for i := 0; i < b.N; i++ {
		gen := generateIPv4()
		m[string(gen)] = true
	}
}
func BenchmarkInsertMapIPv6(b *testing.B) {
	m := make(map[string]bool)
	for i := 0; i < b.N; i++ {
		gen := generateIPv6()
		m[string(gen)] = true
	}
}
func BenchmarkLookupIPv4(b *testing.B) {
	radix := New()

	for i := 0; i < 1000000; i++ {
		gen := generateIPv4()
		radix.Insert(gen.To4(), 64)
	}
	for i := 0; i < b.N; i++ {
		gen := generateIPv4()
		radix.Get(gen.To4())
	}
}
func BenchmarkLookupIPv6(b *testing.B) {
	radix := New()

	for i := 0; i < 1000000; i++ {
		gen := generateIPv6()
		radix.Insert(gen.To16(), 64)
	}

	for i := 0; i < b.N; i++ {
		gen := generateIPv6()
		radix.Get(gen.To16())
	}
}

// generateIPv4 is used to generate a random IP
func generateIPv4() net.IP {
	buf := make([]byte, 4)
	if _, err := crand.Read(buf); err != nil {
		panic(fmt.Errorf("failed to read random bytes: %v", err))
	}
	// let's fix the prefix to emulate a /8 network
	// 2001:0db8:1:2::/64
	buf[0] = 0x10
	return net.IP(buf)
}

// generateIPv6 is used to generate a random IP
func generateIPv6() net.IP {
	buf := make([]byte, 16)
	if _, err := crand.Read(buf); err != nil {
		panic(fmt.Errorf("failed to read random bytes: %v", err))
	}
	// let's fix the prefix to emulate a /64 network
	// 2001:0db8:1:2::/64
	buf[0] = 0x20
	buf[1] = 0x01
	buf[2] = 0x0d
	buf[3] = 0xb8
	buf[4] = 0x00
	buf[5] = 0x01
	buf[6] = 0x00
	buf[7] = 0x02
	return net.IP(buf)
}
