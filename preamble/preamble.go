package preamble

import (
	"github.com/bemasher/fftw"
	"math"
	"math/cmplx"
)

// Preamble detection uses half-complex dft to convolve signal with preamble
// basis function, argmax of result represents most likely preamble position.
type PreambleDetector struct {
	forward  fftw.HCDFT1DPlan
	backward fftw.HCDFT1DPlan

	Real     []float64
	Complex  []complex128
	template []complex128
}

func NewPreambleDetector(n uint, symbolLength float64, bits string) (pd PreambleDetector) {
	// Plan forward and reverse transforms.
	pd.forward = fftw.NewHCDFT1D(n, nil, nil, fftw.Forward, fftw.InPlace, fftw.Measure)
	pd.Real = pd.forward.Real
	pd.Complex = pd.forward.Complex
	pd.backward = fftw.NewHCDFT1D(n, pd.Real, pd.Complex, fftw.Backward, fftw.PreAlloc, fftw.Measure)

	// Zero out input array.
	for i := range pd.Real {
		pd.Real[i] = 0
	}

	// Generate the preamble basis function.
	for idx, bit := range bits {
		// Must account for rounding error.
		sIdx := idx << 1
		lower := IntRound(float64(sIdx) * symbolLength)
		upper := IntRound(float64(sIdx+1) * symbolLength)
		for i := 0; i < upper-lower; i++ {
			if bit == '1' {
				pd.Real[lower+i] = 1.0
				pd.Real[upper+i] = -1.0
			} else {
				pd.Real[lower+i] = -1.0
				pd.Real[upper+i] = 1.0
			}
		}
	}

	// Transform the preamble basis function.
	pd.forward.Execute()

	// Create the preamble template and store conjugated dft result.
	pd.template = make([]complex128, len(pd.Complex))
	copy(pd.template, pd.Complex)
	for i := range pd.template {
		pd.template[i] = cmplx.Conj(pd.template[i])
	}

	return
}

// FFTW plans must be cleaned up.
func (pd *PreambleDetector) Close() {
	pd.forward.Close()
	pd.backward.Close()
}

// Convolves signal with preamble basis function. Returns the most likely
// position of preamble. Assumes data has been copied into real array.
func (pd *PreambleDetector) Execute() int {
	pd.forward.Execute()
	for i := range pd.template {
		pd.backward.Complex[i] = pd.forward.Complex[i] * pd.template[i]
	}
	pd.backward.Execute()

	return pd.ArgMax()
}

// Calculate index of largest element in the real array.
func (pd *PreambleDetector) ArgMax() (idx int) {
	max := 0.0
	for i, v := range pd.backward.Real {
		if max < v {
			max, idx = v, i
		}
	}
	return idx
}

func IntRound(i float64) int {
	return int(math.Floor(i + 0.5))
}
