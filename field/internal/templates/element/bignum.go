package element

const BigNum = `

{{/* Only used for the Pornin Extended GCD Inverse Algorithm*/}}
{{if eq .NoCarry true}}

func (z *{{.ElementName}}) neg(x *{{.ElementName}}, xHi uint64) uint64 {
	var b uint64

	z[0], b = bits.Sub64(0, x[0], 0)
	{{- range $i := .NbWordsIndexesNoZero}}
	z[{{$i}}], b = bits.Sub64(0, x[{{$i}}], b)
	{{- end}}
	xHi, _ = bits.Sub64(0, xHi, b)

	return xHi
}

// regular multiplication by one word regular (non montgomery)
// Fewer additions than the branch-free for positive y. Could be faster on some architectures
func (z *{{.ElementName}}) mulWRegular(x *{{.ElementName}}, y int64) uint64 {

	// w := abs(y)
	m := y >> 63
	w := uint64((y^m)-m)

	var c uint64
	c, z[0] = bits.Mul64(x[0], w)
	{{- range $i := .NbWordsIndexesNoZero }}
	c, z[{{$i}}] = madd1(x[{{$i}}], w, c)
	{{- end}}

	if y < 0 {
		c = z.neg(z, c)
	}

	return c
}

/*
Removed: seems slower
// mulWRegular branch-free regular multiplication by one word (non montgomery)
func (z *{{.ElementName}}) mulWRegularBf(x *{{.ElementName}}, y int64) uint64 {

	w := uint64(y)
	allNeg := uint64(y >> 63)	// -1 if y < 0, 0 o.w

	// s[0], s[1] so results are not stored immediately in z.
	// x[i] will be needed in the i+1 th iteration. We don't want to overwrite it in case x = z
	var s [2]uint64
	var h [2]uint64

	h[0], s[0] = bits.Mul64(x[0], w)

	c := uint64(0)
	b := uint64(0)

	{{- range $i := .NbWordsIndexesNoZero}}

		{
			const curI = {{$i}} % 2
			const prevI = 1 - curI
			const iMinusOne = {{$i}} - 1

			h[curI], s[curI] = bits.Mul64(x[{{$i}}], w)
			s[curI], c = bits.Add64(s[curI], h[prevI], c)
			s[curI], b = bits.Sub64(s[curI], allNeg & x[iMinusOne], b)
			z[iMinusOne] = s[prevI]
		}
	{{- end}}
	{
		const curI = {{.NbWords}} % 2
		const prevI = 1 - curI
		const iMinusOne = {{.NbWordsLastIndex}}

		s[curI], _ = bits.Sub64(h[prevI], allNeg & x[iMinusOne], b)
		z[iMinusOne] = s[prevI]

		return s[curI] + c
	}
}*/

// Requires NoCarry
func (z *{{.ElementName}}) linearCombNonModular(x *{{.ElementName}}, xC int64, y *{{.ElementName}}, yC int64) uint64 {
	var yTimes {{.ElementName}}

	yHi := yTimes.mulWRegular(y, yC)
	xHi := z.mulWRegular(x, xC)

	carry := uint64(0)

	{{- range $i := .NbWordsIndexesFull}}
		z[{{$i}}], carry = bits.Add64(z[{{$i}}], yTimes[{{$i}}], carry)
	{{- end}}

	yHi, _ = bits.Add64(xHi, yHi, carry)

	return yHi
}

{{- end}}
`
