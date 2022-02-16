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

package fptower

import (
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/prop"
	"testing"
)

func TestE12Div(t *testing.T) {

	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)

	genA := GenE12()
	genB := GenE12()

	properties.Property("[BLS12-377] dividing then multiplying by the same element does nothing", prop.ForAll(
		func(a, b *E12) bool {
			var c E12
			c.Div(a, b)
			c.Mul(&c, b)
			return c.Equal(a)
		},
		genA,
		genB,
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
