// Copyright 2020 ConsenSys Software Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by consensys/gnark-crypto DO NOT EDIT

package bls24315

import (
	"github.com/consensys/gnark-crypto/ecc/bls24-315/fp"
	"math/big"
)

// sqrtRatio computes the square root of u/v and returns true if u/v was indeed a quadratic residue
// if not, we get sqrt(Z * u / v). Recall that Z is non-residue
// The return value is undefined for u = 0
// Taken from https://datatracker.ietf.org/doc/draft-irtf-cfrg-hash-to-curve/13/ F.2.1.1. for any field
// The main idea is that since the computation of the square root involves taking large powers of u/v, the inversion of v can be avoided
func sqrtRatio(z *fp.Element, u *fp.Element, v *fp.Element) bool {
	tv1 := fp.Element{11195128742969911322, 1359304652430195240, 15267589139354181340, 10518360976114966361, 300769513466036652}

	var exp big.Int
	exp.SetBytes([]byte{15, 255, 255})
	var tv2, tv3, tv4, tv5 fp.Element
	tv2.Exp(*v, &exp)
	tv3.Mul(&tv2, &tv2)
	tv3.Mul(&tv3, v)

	// line 5
	tv5.Mul(u, &tv3)

	exp.SetBytes([]byte{38, 17, 208, 21, 172, 54, 178, 134, 159, 186, 76, 95, 75, 226, 245, 126, 246, 14, 128, 213, 19, 208, 215, 2, 16, 247, 46, 210, 149, 239, 40, 19, 127, 64, 23, 250, 1})
	tv5.Exp(tv5, &exp)
	tv5.Mul(&tv5, &tv2)
	tv2.Mul(&tv5, v)
	tv3.Mul(&tv5, u)

	// line 10
	tv4.Mul(&tv3, &tv2)
	exp.SetBytes([]byte{8, 0, 0})
	tv5.Exp(tv4, &exp)

	isQr := tv5.IsOne()

	tv2.Mul(&tv3, &fp.Element{1141794007209116247, 256324699145650176, 2958838397954514392, 9976887947641032208, 153331829745922234})
	tv5.Mul(&tv4, &tv1)

	// line 15

	if !isQr {
		tv3, tv4 = tv2, tv5
	}

	exp.Lsh(big.NewInt(1), 20-2)
	for i := 20; i >= 2; i-- {
		//line 20
		tv5.Exp(tv4, &exp)
		e1 := tv5.IsOne()

		tv2.Mul(&tv3, &tv1)
		tv1.Mul(&tv1, &tv1)
		tv5.Mul(&tv4, &tv1)

		if !e1 {
			tv3, tv4 = tv2, tv5
		}

		exp.Rsh(&exp, 1)
	}

	*z = tv3
	return isQr
}

//TODO: Use addchain
//TODO: Might duplicate functionality from mulByConst functions
// mulByZ multiplies x by 13 and stores the result in z
func mulByZ(z *fp.Element, x *fp.Element) {

	res := *x

	res.Double(&res)

	res.Add(&res, x)

	res.Double(&res)

	res.Double(&res)

	res.Add(&res, x)

	*z = res
}

// From https://datatracker.ietf.org/doc/draft-irtf-cfrg-hash-to-curve/13/ Pg 80
func sswuMapG1(u *fp.Element) G1Affine {

	var tv1 fp.Element
	tv1.Square(u)

	//mul tv1 by Z
	mulByZ(&tv1, &tv1)

	var tv2 fp.Element
	tv2.Square(&tv1)
	tv2.Add(&tv2, &tv1)

	var tv3 fp.Element
	//Standard doc line 5
	var tv4 fp.Element
	tv4.SetOne()
	tv3.Add(&tv2, &tv4)
	tv3.Mul(&tv3, &fp.Element{0})

	tv4 = fp.Element{0}
	//TODO: Std doc uses conditional move. If-then-else good enough here?
	if tv2.IsZero() {
		fp.MulBy11(&tv4) //WARNING: this branch takes less time
		//tv4.MulByConstant(Z)
	} else {
		tv4.Mul(&tv4, &tv2)
		tv4.Neg(&tv4)
	}
	tv2.Square(&tv3)

	var tv6 fp.Element
	//Standard doc line 10
	tv6.Square(&tv4)

	var tv5 fp.Element
	tv5.Mul(&tv6, &fp.Element{0})

	tv2.Add(&tv2, &tv5)
	tv2.Mul(&tv2, &tv3)
	tv6.Mul(&tv6, &tv4)

	//Standards doc line 15
	tv5.Mul(&tv6, &fp.Element{0})
	tv2.Add(&tv2, &tv5)

	var x fp.Element
	x.Mul(&tv1, &tv3)

	var y1 fp.Element
	gx1Square := sqrtRatio(&y1, &tv2, &tv6)

	var y fp.Element
	y.Mul(&tv1, u)

	//Standards doc line 20
	y.Mul(&y, &y1)

	//TODO: Not constant time. Is it okay?
	if gx1Square {
		x = tv3
		y = y1
	}

	//TODO: Not constant time
	if u.Sgn0() != y.Sgn0() {
		y.Neg(&y)
	}

	//Standards doc line 25
	//TODO: Not constant time. Use Jacobian?
	x.Div(&x, &tv4)

	return G1Affine{x, y}
}

// EncodeToCurveG1SSWU maps a fp.Element to a point on the curve using the Simplified Shallue and van de Woestijne Ulas map
//https://datatracker.ietf.org/doc/draft-irtf-cfrg-hash-to-curve/13/#section-6.6.3
func EncodeToCurveG1SSWU(msg, dst []byte) (G1Affine, error) {
	var res G1Affine
	t, err := hashToFp(msg, dst, 1)
	if err != nil {
		return res, err
	}
	res = sswuMapG1(&t[0])

	res.ClearCofactor(&res)

	return res, nil
}

// HashToCurveG1SSWU hashes a byte string to the G1 curve. Usable as a random oracle.
// https://tools.ietf.org/html/draft-irtf-cfrg-hash-to-curve-06#section-3
func HashToCurveG1SSWU(msg, dst []byte) (G1Affine, error) {
	var res G1Affine
	u, err := hashToFp(msg, dst, 2)
	if err != nil {
		return res, err
	}

	Q0 := sswuMapG1(&u[0])
	Q1 := sswuMapG1(&u[1])

	var _Q0, _Q1, _res G1Jac
	_Q0.FromAffine(&Q0)
	_Q1.FromAffine(&Q1)
	_res.Set(&_Q1).AddAssign(&_Q0)
	res.FromJacobian(&_res)
	res.ClearCofactor(&res)
	return res, nil
}
