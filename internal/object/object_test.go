package object

import (
	"slug/internal/dec64"
	"testing"
)

func TestStringMapKey(t *testing.T) {
	hello1 := &String{Value: "Hello World"}
	hello2 := &String{Value: "Hello World"}
	diff1 := &String{Value: "My name is johnny"}
	diff2 := &String{Value: "My name is johnny"}

	if hello1.MapKey() != hello2.MapKey() {
		t.Errorf("strings with same content have different map keys")
	}

	if diff1.MapKey() != diff2.MapKey() {
		t.Errorf("strings with same content have different map keys")
	}

	if hello1.MapKey() == diff1.MapKey() {
		t.Errorf("strings with different content have same map keys")
	}
}

func TestBooleanMapKey(t *testing.T) {
	true1 := &Boolean{Value: true}
	true2 := &Boolean{Value: true}
	false1 := &Boolean{Value: false}
	false2 := &Boolean{Value: false}

	if true1.MapKey() != true2.MapKey() {
		t.Errorf("trues do not have same map key")
	}

	if false1.MapKey() != false2.MapKey() {
		t.Errorf("falses do not have same map key")
	}

	if true1.MapKey() == false1.MapKey() {
		t.Errorf("true has same map key as false")
	}
}

func TestIntegerMapKey(t *testing.T) {
	one1 := &Number{Value: dec64.FromInt64(1)}
	one2 := &Number{Value: dec64.FromInt64(1)}
	two1 := &Number{Value: dec64.FromInt64(2)}
	two2 := &Number{Value: dec64.FromInt64(2)}

	if one1.MapKey() != one2.MapKey() {
		t.Errorf("numbers with same content have different map keys")
	}

	if two1.MapKey() != two2.MapKey() {
		t.Errorf("numbers with same content have different map keys")
	}

	if one1.MapKey() == two1.MapKey() {
		t.Errorf("numbers with different content have same map keys, %v : %v", one1, two1)
	}
}

func TestSymbolMapKey(t *testing.T) {
	s1 := InternSymbol("foo")
	s2 := InternSymbol("foo")
	s3 := InternSymbol("bar")

	if s1 != s2 {
		t.Errorf("symbols with same name are not interned")
	}
	if s1.MapKey() != s2.MapKey() {
		t.Errorf("symbols with same name have different map keys")
	}
	if s1.MapKey() == s3.MapKey() {
		t.Errorf("symbols with different names have same map keys")
	}
}
