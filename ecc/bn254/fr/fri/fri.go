package fri

import (
	"bytes"
	"errors"
	"fmt"
	"hash"
	"math/big"
	"math/bits"

	"github.com/consensys/gnark-crypto/accumulator/merkletree"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/fft"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/polynomial"
	fiatshamir "github.com/consensys/gnark-crypto/fiat-shamir"
)

var (
	ErrProximityTest = errors.New("fri proximity test failed")
	ErrOddSize       = errors.New("the size should be even")
)

const rho = 8

// Digest commitment of a polynomial.
type Digest []byte

// merkleProof helper structure to build the merkle proof
type partialMerkleProof struct {
	merkleRoot []byte
	proofSet   [][]byte
	numLeaves  uint64
}

// Iopp interface that an iopp should implement
type Iopp interface {

	// Commit returns the commitment to a polynomial p.
	// The commitment is the root of the Merkle tree corresponding
	// to the Reed Solomon code formed by p.
	// p is not modified after the function call.
	Commit(p polynomial.Polynomial, h hash.Hash) (Digest, error)

	// BuildProofOfProximity creates a proof of proximity that p is d-close to a polynomial
	// of degree len(p). The proof is built non interactively using Fiat Shamir.
	BuildProofOfProximity(p polynomial.Polynomial) (ProofOfProximity, error)

	// VerifyProofOfProximity verifies the proof of proximity. It returns an error if the
	// verification fails.
	VerifyProofOfProximity(proof ProofOfProximity) error
}

// IOPP Interactive Oracle Proof of Proximity
type IOPP uint

const (
	RADIX_2_FRI IOPP = iota
)

// ProofOfProximity proof of proximity, attesting that
// a function is d-close to a low degree polynomial.
type ProofOfProximity struct {

	// stores the interactions between the prover and the verifier.
	// Each interaction results in a set or merkle proofs, corresponding
	// to the queries of the verifier.
	interactions [][]partialMerkleProof

	// evaluation stores the evaluation of the fully folded polynomial.
	// The verifier need to reconstruct the polynomial, and check that
	// it is low degree.
	evaluation []fr.Element
}

// New creates a new IOPP capable to handle degree(size) polynomials.
func (iopp IOPP) New(size uint64, h hash.Hash) Iopp {
	switch iopp {
	case RADIX_2_FRI:
		return newRadixTwoFri(size, h)
	default:
		panic("iopp name is not recognized")
	}
}

// radixTwoFri empty structs implementing compressionFunction for
// the squaring function.
type radixTwoFri struct {

	// hash function that is used for Fiat Shamir and for committing to
	// the oracles.
	h hash.Hash

	// nbSteps number of interactions between the prover and the verifier
	nbSteps int

	// domains list of domains used for fri
	// TODO normally, a single domain of size n=2^c should handle all
	// polynomials of size 2^d where d<c...
	domains []*fft.Domain
}

func newRadixTwoFri(size uint64, h hash.Hash) radixTwoFri {

	var res radixTwoFri

	// computing the number of steps
	n := ecc.NextPowerOfTwo(size)
	nbSteps := bits.TrailingZeros(uint(n))
	res.nbSteps = nbSteps

	// extending the domain
	n = n * rho

	// building the domains
	res.domains = make([]*fft.Domain, nbSteps)
	for i := 0; i < nbSteps; i++ {
		res.domains[i] = fft.NewDomain(n, 0, false)
		n = n >> 1
	}

	// hash function
	res.h = h

	return res
}

// finds i such that g^i = a
// TODO for the moment assume it exits and easily computable
func (s radixTwoFri) log(a, g fr.Element) int {
	var i int
	var _g fr.Element
	_g.SetOne()
	for i = 0; ; i++ {
		if _g.Equal(&a) {
			break
		}
		_g.Mul(&_g, &g)
	}
	return i
}

// convertOrderCanonical convert the index i, an entry in a
// sorted polynomial, to the corresponding entry in canonical
// representation. n is the size of the polynomial.
func convertSortedCanonical(i, n int) int {
	if i%2 == 0 {
		return i / 2
	} else {
		l := (n - 1 - i) / 2
		return n - 1 - l
	}
}

// deriveQueriesPositions derives the indices of the oracle
// function that the verifier has to pick. The result is a
// slice of []int, where each entry is a tuple (i_k), such that
// the verifier needs to evaluate sum_k oracle(i_k)x^k to build
// the folded function.
func (s radixTwoFri) deriveQueriesPositions(a fr.Element) []int {

	res := make([]int, s.nbSteps)

	l := s.log(a, s.domains[0].Generator)
	n := int(s.domains[0].Cardinality)

	// first we convert from canonical indexation to sorted indexation
	for i := 0; i < s.nbSteps; i++ {

		// canonical --> sorted
		if l < n/2 {
			res[i] = 2 * l
		} else {
			res[i] = (n - 1) - 2*(n-1-l)
		}

		if l > n/2 {
			l = l - n/2
		}
		n = n >> 1
	}
	return res
}

// sort orders the evaluation of a polynomial on a domain
// such that contiguous entries are in the same fiber.
func sort(evaluations polynomial.Polynomial) polynomial.Polynomial {
	q := polynomial.New(uint64(len(evaluations)))
	n := len(evaluations) / 2
	for i := 0; i < len(evaluations)/2; i++ {
		q[2*i].Set(&evaluations[i])
		q[2*i+1].Set(&evaluations[i+n])
	}
	return q
}

// Commit returns the commitment to a polynomial p.
// The commitment is the root of the Merkle tree corresponding
// to the Reed Solomon code formed by p.
// p is not modified after the function call.
func (s radixTwoFri) Commit(p polynomial.Polynomial, h hash.Hash) (Digest, error) {

	c := s.domains[0].Cardinality

	_p := polynomial.New(c)
	copy(_p, p)

	s.domains[0].FFT(_p, fft.DIF, 0)
	fft.BitReverse(_p)

	var buf bytes.Buffer

	for i := 0; i < len(_p)/2; i++ {

		// to ease up the query process, that is to minimize the size of the Merkle proof,
		// the oracle stores the evaluations of _p such that contiguous elements belong to
		// the same fiber.
		_, err := buf.Write(_p[i].Marshal())
		if err != nil {
			return nil, err
		}

		_, err = buf.Write(_p[i+len(_p)/2].Marshal())
		if err != nil {
			return nil, err
		}
	}
	tree := merkletree.New(h)
	err := tree.ReadAll(&buf, fr.Bytes)
	if err != nil {
		return nil, err
	}
	return tree.Root(), nil
}

// BuildProofOfProximity generates a proof that a function, given as an oracle from
// the verifier point of view, is in fact d-close to a polynomial.
func (s radixTwoFri) BuildProofOfProximity(p polynomial.Polynomial) (ProofOfProximity, error) {

	extendedSize := int(s.domains[0].Cardinality)
	_p := polynomial.New(uint64(extendedSize))
	copy(_p, p)

	// the proof will contain nbSteps interactions
	var proof ProofOfProximity
	proof.interactions = make([][]partialMerkleProof, s.nbSteps)

	// Fiat Shamir transcript to derive the challenges
	xis := make([]string, s.nbSteps)
	for i := 0; i < s.nbSteps; i++ {
		xis[i] = fmt.Sprintf("x%d", i)
	}
	fs := fiatshamir.NewTranscript(s.h, xis...)

	// step 1 : fold the polynomial using the xi
	evalsAtRound := make([][]fr.Element, s.nbSteps)

	for i := 0; i < s.nbSteps; i++ {

		// evaluate _p and sort the result
		s.domains[i].FFT(_p, fft.DIF, 0)
		fft.BitReverse(_p)
		evalsAtRound[i] = sort(_p)

		// compute the root hash, needed to derive xi
		t := merkletree.New(s.h)
		for k := 0; k < len(evalsAtRound[i]); k++ {
			t.Push(evalsAtRound[i][k].Marshal())
		}
		rh := t.Root()
		err := fs.Bind(xis[i], rh)
		if err != nil {
			return proof, err
		}

		// derive the challenge
		bxi, err := fs.ComputeChallenge(xis[i])
		if err != nil {
			return proof, err
		}
		var xi fr.Element
		xi.SetBytes(bxi)

		// put _p back in canonical basis
		s.domains[i].FFTInverse(_p, fft.DIF, 0)
		fft.BitReverse(_p)

		// fold _p
		fp := polynomial.New(uint64(len(_p) / 2))
		for k := 0; k < len(_p)/2; k++ {
			fp[k].Mul(&_p[2*k+1], &xi)
			fp[k].Add(&fp[k], &_p[2*k])
		}

		_p = fp
	}

	// last round, provide the evaluation
	proof.evaluation = make([]fr.Element, len(_p))
	var g fr.Element
	g.SetOne()
	for i := 0; i < rho; i++ {
		e := _p.Eval(&g)
		proof.evaluation[i].Set(&e)
		g.Mul(&g, &s.domains[s.nbSteps-1].Generator)
	}

	// step 2: provide the Merkle proofs of the queries

	// derive the verifier queries
	// TODO use Fiat Shamir, for the moment take g
	si := s.deriveQueriesPositions(s.domains[0].Generator)

	for i := 0; i < s.nbSteps; i++ {

		// build proofs of queries at s[i]
		t := merkletree.New(s.h)
		err := t.SetIndex(uint64(si[i]))
		if err != nil {
			return proof, err
		}
		for k := 0; k < len(evalsAtRound[i]); k++ {
			t.Push(evalsAtRound[i][k].Marshal())
		}
		mr, proofSet, _, numLeaves := t.Prove()
		proof.interactions[i] = make([]partialMerkleProof, 2)
		c := si[i] % 2
		proof.interactions[i][c] = partialMerkleProof{mr, proofSet, numLeaves}
		proof.interactions[i][1-c] = partialMerkleProof{
			mr,
			make([][]byte, 2),
			numLeaves,
		}
		proof.interactions[i][1-c].proofSet[0] = evalsAtRound[i][si[i]+1-2*c].Marshal()
		s.h.Reset()
		_, err = s.h.Write(proof.interactions[i][c].proofSet[0])
		if err != nil {
			return proof, err
		}
		proof.interactions[i][1-c].proofSet[1] = s.h.Sum(nil)

	}

	return proof, nil
}

// VerifyProofOfProximity verifies the proof of proximity. It returns an error if the
// verification fails.
func (s radixTwoFri) VerifyProofOfProximity(proof ProofOfProximity) error {

	// Fiat Shamir transcript to derive the challenges
	xis := make([]string, s.nbSteps)
	for i := 0; i < s.nbSteps; i++ {
		xis[i] = fmt.Sprintf("x%d", i)
	}
	fs := fiatshamir.NewTranscript(s.h, xis...)
	xi := make([]fr.Element, s.nbSteps)
	for i := 0; i < s.nbSteps; i++ {
		fs.Bind(xis[i], proof.interactions[i][0].merkleRoot)
		bxi, err := fs.ComputeChallenge(xis[i])
		if err != nil {
			return err
		}
		xi[i].SetBytes(bxi)
	}

	// derive the si
	// TODO use FiatShamir
	si := s.deriveQueriesPositions(s.domains[0].Generator)

	var twoInv fr.Element
	twoInv.SetUint64(2).Inverse(&twoInv)

	// for each round check the Merkle proof and the correctness of the folding
	for i := 0; i < len(proof.interactions); i++ {

		// correctness of Merkle proof
		c := si[i] % 2
		res := merkletree.VerifyProof(
			s.h,
			proof.interactions[i][c].merkleRoot,
			proof.interactions[i][c].proofSet,
			uint64(si[i]),
			proof.interactions[i][c].numLeaves,
		)
		if !res {
			return ErrProximityTest
		}

		proofSet := make([][]byte, len(proof.interactions[i][c].proofSet))
		copy(proofSet[2:], proof.interactions[i][c].proofSet[2:])
		proofSet[0] = proof.interactions[i][1-c].proofSet[0]
		proofSet[1] = proof.interactions[i][1-c].proofSet[1]
		res = merkletree.VerifyProof(
			s.h,
			proof.interactions[i][1-c].merkleRoot,
			proofSet,
			uint64(si[i]+1-2*c),
			proof.interactions[i][1-c].numLeaves,
		)
		if !res {
			return ErrProximityTest
		}

		// correctness of the folding
		if i < len(proof.interactions)-1 {

			var fe, fo, l, r, fn fr.Element

			// even part
			l.SetBytes(proof.interactions[i][0].proofSet[0])
			r.SetBytes(proof.interactions[i][1].proofSet[0])
			fe.Add(&l, &r).Mul(&fe, &twoInv)

			// odd part
			m := convertSortedCanonical(si[i], int(s.domains[i].Cardinality))
			bm := big.NewInt(int64(m))
			var ginv fr.Element
			ginv.Set(&s.domains[i].GeneratorInv).Exp(ginv, bm)
			fo.Sub(&l, &r).Mul(&fo, &twoInv).Mul(&fo, &ginv)
			fn.SetBytes(proof.interactions[i+1][si[i+1]%2].proofSet[0])

			// folding
			fo.Mul(&fo, &xi[i]).Add(&fo, &fe)
			if !fo.Equal(&fn) {
				return ErrProximityTest
			}
		}
	}

	// last transition
	var fe, fo, l, r, fn fr.Element

	// even part
	l.SetBytes(proof.interactions[s.nbSteps-1][0].proofSet[0])
	r.SetBytes(proof.interactions[s.nbSteps-1][1].proofSet[0])
	fe.Add(&l, &r).Mul(&fe, &twoInv)

	// odd part
	m := convertSortedCanonical(si[s.nbSteps-1], int(s.domains[s.nbSteps-1].Cardinality))
	bm := big.NewInt(int64(m))
	var ginv fr.Element
	ginv.Set(&s.domains[s.nbSteps-1].GeneratorInv).Exp(ginv, bm)
	fo.Sub(&l, &r).Mul(&fo, &twoInv).Mul(&fo, &ginv)
	_si := convertSortedCanonical(si[s.nbSteps-1], rho)
	fn.Set(&proof.evaluation[_si])

	// folding
	fo.Mul(&fo, &xi[s.nbSteps-1]).Add(&fo, &fe)
	if !fo.Equal(&fn) {
		return ErrProximityTest
	}

	// Last step: check that the evatuations lie on a line
	dx := make([]fr.Element, rho-1)
	dy := make([]fr.Element, rho-1)
	var _g, g, one fr.Element
	g.Set(&s.domains[s.nbSteps-1].Generator).Square(&g)
	_g.Set(&g)
	one.SetOne()
	for i := 1; i < rho; i++ {
		dx[i-1].Sub(&g, &one)
		dy[i-1].Sub(&proof.evaluation[i], &proof.evaluation[i-1])
		g.Mul(&g, &_g)
	}
	dx = fr.BatchInvert(dx)
	dydx := make([]fr.Element, rho-1)
	for i := 1; i < rho; i++ {
		dydx[i-1].Mul(&dy[i-1], &dx[i-1])
	}

	for i := 1; i < rho-1; i++ {
		if !dydx[0].Equal(&dydx[i]) {
			return ErrProximityTest
		}
	}

	return nil
}