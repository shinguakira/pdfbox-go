package cos

import "testing"

// Ported from pdfbox/src/test/java/org/apache/pdfbox/cos/TestCOSArray.java.

func TestArrayBaseContract(t *testing.T) {
	assertBaseContract(t, NewArray())
}

func TestArrayAccept(t *testing.T) {
	assertVisits(t, NewArray(), "array")
}

func TestArrayCreate(t *testing.T) {
	a := NewArray()
	if got := a.Size(); got != 0 {
		t.Errorf("Size() = %d, want 0", got)
	}

	// Java asserts NullPointerException for a null list; a nil slice is the Go
	// equivalent input, and an empty array is the sensible result rather than
	// a panic.
	if got := NewArrayOf(nil).Size(); got != 0 {
		t.Errorf("NewArrayOf(nil).Size() = %d, want 0", got)
	}

	a = NewArrayOf([]Base{A, B, C})
	if got := a.Size(); got != 3 {
		t.Fatalf("Size() = %d, want 3", got)
	}
	for i, want := range []*Name{A, B, C} {
		if got := a.Get(i); got != Base(want) {
			t.Errorf("Get(%d) = %v, want %v", i, got, want)
		}
	}
}

func TestArrayConvertNames(t *testing.T) {
	a := ArrayOfNames([]string{A.Name(), B.Name(), C.Name()})
	if got := a.Size(); got != 3 {
		t.Fatalf("Size() = %d, want 3", got)
	}
	for i, want := range []*Name{A, B, C} {
		if got := a.Get(i); got != Base(want) {
			t.Errorf("Get(%d) = %v, want %v", i, got, want)
		}
	}

	list := a.ToNameStringList()
	if len(list) != 3 {
		t.Fatalf("ToNameStringList() length = %d, want 3", len(list))
	}
	for i, want := range []string{A.Name(), B.Name(), C.Name()} {
		if list[i] == nil || *list[i] != want {
			t.Errorf("ToNameStringList()[%d] = %v, want %q", i, list[i], want)
		}
	}
}

func TestArrayConvertStrings(t *testing.T) {
	a := ArrayOfStrings([]string{"A", "B", "C"})
	if got := a.Size(); got != 3 {
		t.Fatalf("Size() = %d, want 3", got)
	}
	for i, want := range []string{"A", "B", "C"} {
		if got := a.GetString(i, ""); got != want {
			t.Errorf("GetString(%d) = %q, want %q", i, got, want)
		}
	}

	list := a.ToStringStringList()
	if len(list) != 3 {
		t.Fatalf("ToStringStringList() length = %d, want 3", len(list))
	}
	for i, want := range []string{"A", "B", "C"} {
		if list[i] == nil || *list[i] != want {
			t.Errorf("ToStringStringList()[%d] = %v, want %q", i, list[i], want)
		}
	}
}

func TestArrayConvertIntegers(t *testing.T) {
	a := ArrayOfIntegers([]int{1, 2, 3})
	if got := a.Size(); got != 3 {
		t.Fatalf("Size() = %d, want 3", got)
	}
	for i, want := range []int{1, 2, 3} {
		if got := a.GetInt(i); got != want {
			t.Errorf("GetInt(%d) = %d, want %d", i, got, want)
		}
	}

	list := a.ToNumberIntegerList()
	if len(list) != 3 {
		t.Fatalf("ToNumberIntegerList() length = %d, want 3", len(list))
	}
	for i, want := range []int{1, 2, 3} {
		if list[i] == nil || *list[i] != want {
			t.Errorf("ToNumberIntegerList()[%d] = %v, want %d", i, list[i], want)
		}
	}

	// arrays holding a nil entry
	a = NewArrayOf([]Base{GetInteger(1), nil, GetInteger(3)})
	if got := a.Size(); got != 3 {
		t.Fatalf("Size() = %d, want 3", got)
	}
	if got := a.GetInt(0); got != 1 {
		t.Errorf("GetInt(0) = %d, want 1", got)
	}
	if got := a.Get(1); got != nil {
		t.Errorf("Get(1) = %v, want nil", got)
	}
	if got := a.GetInt(2); got != 3 {
		t.Errorf("GetInt(2) = %d, want 3", got)
	}

	list = a.ToNumberIntegerList()
	if len(list) != 3 {
		t.Fatalf("ToNumberIntegerList() length = %d, want 3", len(list))
	}
	if list[0] == nil || *list[0] != 1 {
		t.Errorf("[0] = %v, want 1", list[0])
	}
	if list[1] != nil {
		t.Errorf("[1] = %v, want nil", list[1])
	}
	if list[2] == nil || *list[2] != 3 {
		t.Errorf("[2] = %v, want 3", list[2])
	}
}

func TestArrayConvertFloats(t *testing.T) {
	start := []float32{1, 0.1, 0.02}
	a := NewArray()
	a.SetFloatArray(start)

	if got := a.Size(); got != 3 {
		t.Fatalf("Size() = %d, want 3", got)
	}
	if got := a.Get(0); !got.(*Float).Equals(FloatOne) {
		t.Errorf("Get(0) = %v, want 1", got)
	}
	if got := a.Get(1); !got.(*Float).Equals(NewFloat(0.1)) {
		t.Errorf("Get(1) = %v, want 0.1", got)
	}

	list := a.ToNumberFloatList()
	if len(list) != 3 {
		t.Fatalf("ToNumberFloatList() length = %d, want 3", len(list))
	}
	for i, want := range start {
		if list[i] == nil || *list[i] != want {
			t.Errorf("ToNumberFloatList()[%d] = %v, want %v", i, list[i], want)
		}
	}

	end := a.ToFloatArray()
	if len(end) != len(start) {
		t.Fatalf("ToFloatArray() length = %d, want %d", len(end), len(start))
	}
	for i := range start {
		if end[i] != start[i] {
			t.Errorf("ToFloatArray()[%d] = %v, want %v", i, end[i], start[i])
		}
	}

	// a nil entry becomes 0 in the float array
	a = NewArrayOf([]Base{FloatOne, nil, NewFloat(0.02)})
	if got := a.Size(); got != 3 {
		t.Fatalf("Size() = %d, want 3", got)
	}
	if got := a.Get(1); got != nil {
		t.Errorf("Get(1) = %v, want nil", got)
	}
	end = a.ToFloatArray()
	for i, want := range []float32{1, 0, 0.02} {
		if end[i] != want {
			t.Errorf("ToFloatArray()[%d] = %v, want %v", i, end[i], want)
		}
	}
}

func TestArrayGetSetName(t *testing.T) {
	a := NewArray()
	a.GrowToSize(3)
	a.SetName(0, "A")
	a.SetName(1, "B")
	a.SetName(2, "C")

	if got := a.Size(); got != 3 {
		t.Fatalf("Size() = %d, want 3", got)
	}
	for i, want := range []string{"A", "B", "C"} {
		if got := a.GetName(i, ""); got != want {
			t.Errorf("GetName(%d) = %q, want %q", i, got, want)
		}
	}
	if got := a.GetName(3, "NULL"); got != "NULL" {
		t.Errorf("GetName(3, NULL) = %q, want NULL", got)
	}

	for i, n := range []*Name{A, B, C} {
		if got := a.IndexOf(n); got != i {
			t.Errorf("IndexOf(%v) = %d, want %d", n, got, i)
		}
	}
	if got := a.IndexOf(D); got != -1 {
		t.Errorf("IndexOf(D) = %d, want -1", got)
	}

	a.SetName(1, "D")
	if got := a.Size(); got != 3 {
		t.Errorf("Size() = %d after SetName, want 3", got)
	}
	if got := a.GetName(1, ""); got != "D" {
		t.Errorf("GetName(1) = %q, want D", got)
	}
}

func TestArrayGetSetInt(t *testing.T) {
	a := NewArray()
	a.GrowToSize(3)
	a.SetInt(0, 0)
	a.SetInt(1, 1)
	a.SetInt(2, 2)

	if got := a.Size(); got != 3 {
		t.Fatalf("Size() = %d, want 3", got)
	}
	for i := 0; i < 3; i++ {
		if got := a.GetInt(i); got != i {
			t.Errorf("GetInt(%d) = %d, want %d", i, got, i)
		}
	}
	if got := a.GetIntDefault(3, 0); got != 0 {
		t.Errorf("GetIntDefault(3, 0) = %d, want 0", got)
	}

	for i := 0; i < 3; i++ {
		if got := a.IndexOf(GetInteger(int64(i))); got != i {
			t.Errorf("IndexOf(%d) = %d, want %d", i, got, i)
		}
	}
	if got := a.IndexOf(GetInteger(3)); got != -1 {
		t.Errorf("IndexOf(3) = %d, want -1", got)
	}

	a.SetInt(1, 3)
	if got := a.GetInt(1); got != 3 {
		t.Errorf("GetInt(1) = %d, want 3", got)
	}
}

func TestArrayGetSetString(t *testing.T) {
	a := NewArray()
	a.GrowToSize(3)
	a.SetString(0, "Test1")
	a.SetString(1, "Test2")
	a.SetString(2, "Test3")

	if got := a.Size(); got != 3 {
		t.Fatalf("Size() = %d, want 3", got)
	}
	for i, want := range []string{"Test1", "Test2", "Test3"} {
		if got := a.GetString(i, ""); got != want {
			t.Errorf("GetString(%d) = %q, want %q", i, got, want)
		}
	}
	if got := a.GetString(3, "NULL"); got != "NULL" {
		t.Errorf("GetString(3, NULL) = %q, want NULL", got)
	}

	for i, want := range []string{"Test1", "Test2", "Test3"} {
		if got := a.IndexOf(NewStringObj(want)); got != i {
			t.Errorf("IndexOf(%q) = %d, want %d", want, got, i)
		}
	}
	if got := a.IndexOf(NewStringObj("Test4")); got != -1 {
		t.Errorf("IndexOf(Test4) = %d, want -1", got)
	}

	a.SetString(1, "Test4")
	if got := a.GetString(1, ""); got != "Test4" {
		t.Errorf("GetString(1) = %q, want Test4", got)
	}
}

func TestArrayRemove(t *testing.T) {
	a := ArrayOfIntegers([]int{1, 2, 3, 4, 5, 6})
	a.Clear()
	if got := a.Size(); got != 0 {
		t.Errorf("Size() = %d after Clear, want 0", got)
	}

	a = ArrayOfIntegers([]int{1, 2, 3, 4, 5, 6})
	if got := a.RemoveAt(2); got != Base(GetInteger(3)) {
		t.Errorf("RemoveAt(2) = %v, want 3", got)
	}
	// 1,2,4,5,6 left
	if got := a.Size(); got != 5 {
		t.Fatalf("Size() = %d, want 5", got)
	}
	if got := a.GetInt(0); got != 1 {
		t.Errorf("GetInt(0) = %d, want 1", got)
	}
	if got := a.GetInt(2); got != 4 {
		t.Errorf("GetInt(2) = %d, want 4", got)
	}

	// 1,2,4,6 left
	if !a.RemoveObject(GetInteger(5)) {
		t.Error("RemoveObject(5) = false, want true")
	}
	if got := a.Size(); got != 4 {
		t.Fatalf("Size() = %d, want 4", got)
	}
	if got := a.GetInt(3); got != 6 {
		t.Errorf("GetInt(3) = %d, want 6", got)
	}

	a = ArrayOfIntegers([]int{1, 2, 3, 4, 5, 6})
	a.RemoveAll([]Base{GetInteger(3), GetInteger(4)})
	// 1,2,5,6 left
	if got := a.Size(); got != 4 {
		t.Fatalf("Size() = %d, want 4", got)
	}
	if got := a.GetInt(1); got != 2 {
		t.Errorf("GetInt(1) = %d, want 2", got)
	}
	if got := a.GetInt(2); got != 5 {
		t.Errorf("GetInt(2) = %d, want 5", got)
	}

	a = ArrayOfIntegers([]int{1, 2, 3, 4, 5, 6})
	a.RetainAll([]Base{GetInteger(3), GetInteger(4)})
	// 3,4 left
	if got := a.Size(); got != 2 {
		t.Fatalf("Size() = %d, want 2", got)
	}
	if got := a.GetInt(0); got != 3 {
		t.Errorf("GetInt(0) = %d, want 3", got)
	}
	if got := a.GetInt(1); got != 4 {
		t.Errorf("GetInt(1) = %d, want 4", got)
	}
}

func TestArrayGrowToSize(t *testing.T) {
	a := NewArray()
	if got := a.Size(); got != 0 {
		t.Fatalf("Size() = %d, want 0", got)
	}

	a.GrowToSize(2)
	// two empty elements
	if got := a.Size(); got != 2 {
		t.Fatalf("Size() = %d, want 2", got)
	}

	// already that size, so nothing happens
	a.GrowToSizeWith(2, GetInteger(0))
	if got := a.Size(); got != 2 {
		t.Fatalf("Size() = %d, want 2", got)
	}

	// grow, filling the new elements
	a.GrowToSizeWith(4, GetInteger(1))
	if got := a.Size(); got != 4 {
		t.Fatalf("Size() = %d, want 4", got)
	}
	list := a.ToNumberIntegerList()
	if len(list) != 4 {
		t.Fatalf("ToNumberIntegerList() length = %d, want 4", len(list))
	}
	if list[0] != nil {
		t.Errorf("[0] = %v, want nil", list[0])
	}
	if list[2] == nil || *list[2] != 1 {
		t.Errorf("[2] = %v, want 1", list[2])
	}
	if list[3] == nil || *list[3] != 1 {
		t.Errorf("[3] = %v, want 1", list[3])
	}
}

func TestArrayToList(t *testing.T) {
	a := ArrayOfIntegers([]int{0, 1, 2, 3, 4, 5})
	list := a.ToList()
	if len(list) != 6 {
		t.Fatalf("ToList() length = %d, want 6", len(list))
	}
	if list[0] != Base(GetInteger(0)) {
		t.Errorf("[0] = %v, want 0", list[0])
	}
	if list[5] != Base(GetInteger(5)) {
		t.Errorf("[5] = %v, want 5", list[5])
	}
}

// TestArrayGetObjectDereferences pins the difference between Get and GetObject:
// Get hands back the raw entry, GetObject resolves an indirect reference and
// maps the null object to nil.
func TestArrayGetObjectDereferences(t *testing.T) {
	inner := GetPDFName("Resolved")
	a := NewArrayOf([]Base{NewObject(inner), NewObject(NullObject)})

	if _, ok := a.Get(0).(*Object); !ok {
		t.Errorf("Get(0) = %T, want the raw *Object", a.Get(0))
	}
	if got := a.GetObject(0); got != Base(inner) {
		t.Errorf("GetObject(0) = %v, want the resolved name", got)
	}
	if got := a.GetObject(1); got != nil {
		t.Errorf("GetObject(1) = %v, want nil for the null object", got)
	}
}
