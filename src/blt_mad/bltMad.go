package blt_mad

import (
	"log"
	"math"
)

//Mad
func Mad(data []float64) float64 {
	//implement MAD given the list of the number of messages in each given bucket
	sum := 0.0
	if len(data) == 0 {
		return math.SmallestNonzeroFloat64 //invalid Mad in terms of BGP message counts
	} else {
		for _, value := range data {
			sum += math.Abs(value - findMean(data))
		}
	}
	result := sum / float64(len(data))
	return result
}

//only neeed for optimization
func BltMad(data []float64, tau float64) []float64 {
	var outliers []float64
	noZeroData, err := RemoveZeros(data)
	if err != nil {
		log.Fatal(err)
	} else {
		//calculate blt formula
		//everything is based on the noZeroData since we are looking at spikes rather than lack of messages
		med := FindMedian(noZeroData)
		m := Mad(noZeroData)

		bltScore := math.Abs(med - tau*m)
		for _, value := range noZeroData {
			if value > bltScore {
				outliers = append(outliers, value)
			}
		}
	}

	return outliers
}

func IsAnOutlierBLT(data []float64, tau float64, point float64) bool {
	//calculate blt formula
	//everything is based on the noZeroData since we are looking at spikes rather than lack of messages
	noZeroData, err := RemoveZeros(data)
	if err != nil {
		log.Fatal(err)
	} else {
		med := FindMedian(noZeroData)
		m := Mad(noZeroData)

		bltScore := math.Abs(med - tau*m)
		if point > bltScore {
			return true
		}
	}
	return false
}
