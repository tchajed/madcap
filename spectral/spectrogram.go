package spectral

import (
	"github.com/mjibson/go-dsp/fft"
	"github.com/mjibson/go-dsp/window"
	"math"
	"math/cmplx"
)

type Spectrogram struct {
	spectra [][]complex128
	LogFreq []float64
}

// Compute the spectrogram of a signal with window width nfft and overlap
// percentage overlap.
func Compute(signal []float64, nfft int, overlap float64) Spectrogram {
	spectra := make([][]complex128, 0)
	start := 0
	end := start + nfft
	off := int((1 - overlap) * float64(nfft))
	// pre-compute the window function
	window := window.Hamming(nfft)
	// pre-allocate buffer to hold window * signal
	windowed := make([]float64, nfft)
	for end < len(signal) {
		for i, s := range signal[start:end] {
			windowed[i] = s * window[i]
		}
		spectrum := fft.FFTReal(windowed)
		// FIXME: memory is still used for the entire spectrum since the GC doesn't
		// understand and can't free internal and partial pointers, to free it must
		// do a copy. If this spectrum persists for a while then we should do a
		// copy().
		spectra = append(spectra, spectrum[0:len(spectrum)/2])
		start += off
		end += off
	}
	spectrogram := Spectrogram{spectra, nil}
	return spectrogram
}

// utility to keep track of basic sample statistics online
type dataset struct {
	sum             float64
	sumSquaredError float64
	n               int
	maxval          float64
	maxindex        int
}

func (d *dataset) Sum() float64 {
	return d.sum
}

func (d *dataset) N() int {
	return d.n
}

func (d *dataset) Mean() float64 {
	return d.Sum() / float64(d.N())
}

func (d *dataset) Variance() float64 {
	return d.sumSquaredError / float64(d.N()-1)
}

func (d *dataset) StdDev() float64 {
	return math.Sqrt(d.Variance())
}

func (d *dataset) CoefficientOfVariation() float64 {
	return d.Mean() / d.StdDev()
}

func (d *dataset) Max() (index int, val float64) {
	return d.maxindex, d.maxval
}

func (d *dataset) Add(x float64) {
	oldmean := d.Mean()
	d.sum += x
	d.n++
	if d.n > 1 {
		d.sumSquaredError += (x - oldmean) * (x - d.Mean())
	}
	if d.n == 1 || x > d.maxval {
		d.maxval = x
		d.maxindex = d.n - 1
	}
}

// scale an index to a frequency, using a particular sample rate sr
func (s *Spectrogram) Frequency(i, sr int) float64 {
	// no spectrum, can't scale appropriately
	if len(s.spectra) == 0 {
		return 0
	}
	// half the sample rate corresponds to the last index of the spectrum
	return float64(sr) / float64(2) * float64(i+1) / float64(len(s.spectra[0]))
}

// Compute spectral statistics for this spectrogram with a given sample rate sr,
// which should be in samples/sec (Hz). Returns a few sample statistics and also
// populates s.LogFreq, energy at logarithmically spaced frequency bins
func (s *Spectrogram) Stats(sr int) map[string]float64 {
	const nfft = 1024
	var (
		cutoffFreq dataset // 80th percentile (in energy) frequency
		energy     dataset // energy of entire spectrum
	)
	frequencyEnergies := make([]dataset, nfft)
	for _, spectrum := range s.spectra {
		var totalEnergy float64
		for fi, fhat := range spectrum {
			frequencyEnergies[fi].Add(cmplx.Abs(fhat))
			totalEnergy += real(fhat * cmplx.Conj(fhat))
		}
		energy.Add(totalEnergy)
		// compute the 80th percentile frequency
		var cumEnergy float64
		for fi, fhat := range spectrum {
			cumEnergy += real(fhat * cmplx.Conj(fhat))
			if cumEnergy >= totalEnergy*0.8 {
				// add the scaled frequency
				cutoffFreq.Add(s.Frequency(fi, sr))
				break
			}
		}
	}
	var freqVariances dataset
	var freqEnergies dataset
	for _, energySet := range frequencyEnergies {
		freqVariances.Add(energySet.Variance())
		freqEnergies.Add(energySet.Mean())
	}
	maxVarFreq, maxVarVal := freqVariances.Max()
	maxEnergyFreq, maxEnergyVal := freqEnergies.Max()
	statistics := make(map[string]float64)
	// cutoff frequency, averaged over time (defined as 80th percentile)
	statistics["cutoffFreq"] = cutoffFreq.Mean()
	// coefficient of variation of energy among the spectra
	statistics["energyCV"] = energy.CoefficientOfVariation()
	// frequency with maximal variance
	statistics["maxVarFreq"] = s.Frequency(maxVarFreq, sr)
	statistics["maxVarVal"] = maxVarVal
	// frequency with the most energy
	statistics["maxEnergyFreq"] = s.Frequency(maxEnergyFreq, sr)
	statistics["maxEnergyVal"] = maxEnergyVal
	frequencyBins := [...]float64{100, 200, 300, 400, 600, 1000, 2000, 4000, 5000, 10000}
	logFreqEnergySets := make([]dataset, len(frequencyBins)+1)
	currIdx := 0
	for i, spectrum := range s.spectra {
		if currIdx < len(frequencyBins) && s.Frequency(i, sr) > frequencyBins[currIdx] {
			currIdx++
		}
		var totalEnergy float64
		for _, fhat := range spectrum {
			totalEnergy += real(fhat * cmplx.Conj(fhat))
		}
		logFreqEnergySets[currIdx].Add(totalEnergy)
	}
	logFreqEnergy := make([]float64, len(logFreqEnergySets))
	for i, set := range logFreqEnergySets {
		logFreqEnergy[i] = set.Sum()
	}
	s.LogFreq = logFreqEnergy

	return statistics
}
