// Code generated by internal/tower DO NOT EDIT
package bn256

import (
	"github.com/consensys/gurvy/bn256/fp"
	"reflect"
	"testing"
)

type E2TestPoint struct {
	in  [2]E2
	out [7]E2
}

var E2TestPoints []E2TestPoint

// TODO this method is the same everywhere. move it someplace central and call it "compare"
func E2compare(t *testing.T, got, want interface{}) {
	if !reflect.DeepEqual(got, want) {
		t.Fatal("\nexpect:\t", want, "\ngot:\t", got)
	}
}

func E2check(t *testing.T, f func(*E2, *E2, *E2) *E2, m int) {

	if len(E2TestPoints) < 1 {
		t.Log("no tests to run")
	}

	for i := range E2TestPoints {
		var receiver E2
		var out *E2
		var inCopies [len(E2TestPoints[i].in)]E2

		for j := range inCopies {
			inCopies[j].Set(&E2TestPoints[i].in[j])
		}

		// receiver, return value both set to result
		out = f(&receiver, &inCopies[0], &inCopies[1])

		E2compare(t, receiver, E2TestPoints[i].out[m]) // receiver correct
		E2compare(t, *out, E2TestPoints[i].out[m])     // return value correct
		for j := range inCopies {
			E2compare(t, inCopies[j], E2TestPoints[i].in[j]) // inputs unchanged
		}

		// receiver == one of the inputs
		for j := range inCopies {
			out = f(&inCopies[j], &inCopies[0], &inCopies[1])

			E2compare(t, inCopies[j], E2TestPoints[i].out[m]) // receiver correct
			E2compare(t, *out, E2TestPoints[i].out[m])        // return value correct
			for k := range inCopies {
				if k == j {
					continue
				}
				E2compare(t, inCopies[k], E2TestPoints[i].in[k]) // other inputs unchanged
			}
			inCopies[j].Set(&E2TestPoints[i].in[j]) // reset input for next tests
		}
	}
}

//--------------------//
//     tests		  //
//--------------------//

func TestE2Add(t *testing.T) {
	E2check(t, (*E2).Add, 0)
}

func TestE2Sub(t *testing.T) {
	E2check(t, (*E2).Sub, 1)
}

func TestE2Mul(t *testing.T) {
	E2check(t, (*E2).Mul, 2)
}

func TestE2MulByElement(t *testing.T) {
	E2check(t, (*E2).MulByElementBinary, 3)
}

func TestE2Square(t *testing.T) {
	E2check(t, (*E2).SquareBinary, 4)
}

func TestE2Inverse(t *testing.T) {
	E2check(t, (*E2).InverseBinary, 5)
}

func TestE2Conjugate(t *testing.T) {
	E2check(t, (*E2).ConjugateBinary, 6)
}

//--------------------//
//     benches		  //
//--------------------//

var E2BenchIn1, E2BenchIn2, E2BenchOut E2

func BenchmarkE2Add(b *testing.B) {
	var a, c E2
	a.SetRandom()
	c.SetRandom()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Add(&a, &c)
	}
}

func BenchmarkE2Sub(b *testing.B) {
	var a, c E2
	a.SetRandom()
	c.SetRandom()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Sub(&a, &c)
	}
}

func BenchmarkE2Mul(b *testing.B) {
	var a, c E2
	a.SetRandom()
	c.SetRandom()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Mul(&a, &c)
	}
}

func BenchmarkE2MulByElement(b *testing.B) {
	var a E2
	var c fp.Element
	c.SetRandom()
	a.SetRandom()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.MulByElement(&a, &c)
	}
}

func BenchmarkE2Square(b *testing.B) {
	var a E2
	a.SetRandom()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Square(&a)
	}
}

func BenchmarkE2Inverse(b *testing.B) {
	var a E2
	a.SetRandom()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Inverse(&a)
	}
}

func BenchmarkE2MulNonRes(b *testing.B) {
	var a E2
	a.SetRandom()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.MulByNonResidue(&a)
	}
}

func BenchmarkE2MulNonResInv(b *testing.B) {
	var a E2
	a.SetRandom()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.MulByNonResidueInv(&a)
	}
}

func BenchmarkE2Conjugate(b *testing.B) {
	var a E2
	a.SetRandom()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Conjugate(&a)
	}
}

//-------------------------------------//
// unary helpers for E2 methods
//-------------------------------------//

// SquareBinary a binary wrapper for Square
func (z *E2) SquareBinary(x, y *E2) *E2 {
	return z.Square(x)
}

// InverseBinary a binary wrapper for Inverse
func (z *E2) InverseBinary(x, y *E2) *E2 {
	return z.Inverse(x)
}

// ConjugateBinary a binary wrapper for Conjugate
func (z *E2) ConjugateBinary(x, y *E2) *E2 {
	return z.Conjugate(x)
}

//-------------------------------------//
// custom helpers for E2 methods
//-------------------------------------//

// MulByElementBinary a binary wrapper for MulByElement
func (z *E2) MulByElementBinary(x, y *E2) *E2 {
	return z.MulByElement(x, &y.A0)
}
