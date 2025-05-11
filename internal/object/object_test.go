package object

import "testing"

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
	one1 := &Integer{Value: 1}
	one2 := &Integer{Value: 1}
	two1 := &Integer{Value: 2}
	two2 := &Integer{Value: 2}

	if one1.MapKey() != one2.MapKey() {
		t.Errorf("integers with same content have different map keys")
	}

	if two1.MapKey() != two2.MapKey() {
		t.Errorf("integers with same content have different map keys")
	}

	if one1.MapKey() == two1.MapKey() {
		t.Errorf("integers with different content have same map keys")
	}
}
