package analyze

import (
	"BGPAlert/blt_mad"
	"BGPAlert/common"
	"BGPAlert/config"
	"BGPAlert/shake_alert"
	"fmt"
	"path/filepath"
	"sort"
	"time"
)

// need to create a map of results based on filter
var fullResultFile, _ = filepath.Abs("fullResult.json")

// var AllResults common.Result //global so that it can be added onto by parser and seen by dataHandler
var maxPoints = 20 //the max number of points to be displayed on the graph;
var optParam = 5.0
var ResultMap = make(map[string]*common.Result)

const (
	BLTMAD     = 1
	SHAKEALERT = 2
	BOTHALERTS = 3
)

// make sure to divide this by the number
//of points in each window or specify that maxPoints is the number of buckets that will be processed

// Takes in a Window, parses object into frequency counts, and then calls specified analysis functions
// code to write the frequencies; the outliers; and the minReqs to files
func AnalyzeBGPMessages(window common.Window, config *config.Configuration) {
	// Get current result from map if it exists
	currResult, exists := ResultMap[window.Filter]

	// If the result is not already in the map, create a new result for it
	if !exists {
		result := common.Result{
			Filter:      window.Filter,
			AllOutliers: make(map[time.Time]common.OutlierInfo),
			AllFreq:     make(map[time.Time]float64),
		}

		// Add window to the map
		ResultMap[window.Filter] = &result

		// Update currWindow to point to the newly added window
		currResult = &result
	}

	lengthMap := makeLengthMap(window)
	// Turn map into sorted array of frequencies by timestamp
	sortedFrequencies, sortedTimestamps := GetSortedFrequencies(lengthMap) //fix these duplicates with time stamps
	frequencies := currResult.AllFreq

	//the file names will contain all the timestamps for a given folder that was processed
	fmt.Printf("Sorted Array of Frequencies: \n%+v\n", sortedFrequencies)
	bltOutliers, bltIndexes := blt_mad.BltMad(sortedFrequencies, optParam)
	shakeAlertOutliers, shakeAlertIndexes := shake_alert.FindOutliers(sortedFrequencies)
	fmt.Printf("BLT MAD Outliers: \n%+v\n", bltOutliers)
	fmt.Printf("ShakeAlert Outliers: \n%+v\n", shakeAlertOutliers)

	//modify the frequencies for the final results
	for timestamp, _ := range lengthMap {
		_, resultVal := frequencies[timestamp]
		if resultVal { //if there exists a value at a given time stamp in the final results
			if frequencies[timestamp] < lengthMap[timestamp] { //check if the existing value is smaller than the incoming
				frequencies[timestamp] = lengthMap[timestamp]
			} else {
				continue //if smaller append onto map else skip
			}
		} else { //the timestamp is not in the results so append
			frequencies[timestamp] = lengthMap[timestamp]
		}
	}

	// Update outliers for both BLT MAD and ShakeAlert outliers
	updateOutliers(currResult, BLTMAD, bltIndexes, sortedTimestamps, sortedFrequencies)
	updateOutliers(currResult, SHAKEALERT, shakeAlertIndexes, sortedTimestamps, sortedFrequencies)

	// Store results into json
	blt_mad.StoreResultIntoJson(currResult, fullResultFile) //storing the most recent result
}

func updateOutliers(currResult *common.Result, algorithm int, indexes []int, sortedTimestamps []time.Time, sortedFrequencies []float64) {
	for _, index := range indexes {
		// Get outlier timestamp
		outlierTimestamp := sortedTimestamps[index]
		outlierFrequency := sortedFrequencies[index]

		// Go through current outliers and see if timestamp exists
		currOutlier, exists := currResult.AllOutliers[outlierTimestamp]

		// If the outlier is not already in the map, create a new outlier for it
		if !exists {
			currResult.AllOutliers[outlierTimestamp] = createOutlierStruct(outlierTimestamp, algorithm, outlierFrequency)
		} else { // If outlier not in map, need to possibly update it
			if currOutlier.Count != outlierFrequency {
				currOutlier.Count = outlierFrequency
			}

			// Update algorithm based on conditions
			switch {
			case currOutlier.Algorithm != BOTHALERTS:
				if algorithm == BLTMAD && currOutlier.Algorithm == SHAKEALERT ||
					algorithm == SHAKEALERT && currOutlier.Algorithm == BLTMAD {
					currOutlier.Algorithm = BOTHALERTS
				} else {
					currOutlier.Algorithm = algorithm
				}
			}

			// Store the updated outlier back in the map
			currResult.AllOutliers[outlierTimestamp] = currOutlier
		}
	}
}

func containsVal(times []time.Time, specificTime time.Time) bool {
	for _, val := range times {
		if val == specificTime { //check equal here
			return true
		}
	}
	return false
}

// there must be an easier way to do this
func getListTimes(outliers []common.OutlierInfo) []time.Time {
	times := []time.Time{}
	for _, val := range outliers {
		times = append(times, val.Timestamp)
	}
	return times
}

// for a given window check what the outliers are and record them into a list of structs
// does not insert an algorithm if both determined it???????????????????????????????????????????????????
// record all the values without repeat in the frequencies :(((((((((((((((((((((((((((((((((((((
func createOutliers(lengthMap map[time.Time]float64) []common.OutlierInfo {
	windowOutliers := []common.OutlierInfo{}
	sortedFrequencies, _ := GetSortedFrequencies(lengthMap)

	//check if each individual entry in the length map is an outlier and create an outlier struct if needed
	for timestamp, _ := range lengthMap {
		//blt outlier; this works im p sure
		if blt_mad.IsAnOutlierBLT(sortedFrequencies, optParam, lengthMap[timestamp]) {
			oMad := createOutlierStruct(timestamp, -1, lengthMap[timestamp])
			windowOutliers = append(windowOutliers, oMad)
			//shakeOutlier
		} else if shake_alert.IsAnOutlierShakeAlert(sortedFrequencies, lengthMap[timestamp]) {
			oShake := createOutlierStruct(timestamp, 1, lengthMap[timestamp])
			fmt.Println(oShake)
			//windowOutliers = append(windowOutliers, oShake)
		} else { //not an outlier
			continue
		}
	}
	return windowOutliers
}

func createOutlierStruct(timestamp time.Time, algorithm int, count float64) common.OutlierInfo {
	o := common.OutlierInfo{
		Timestamp: timestamp,
		Algorithm: algorithm,
		Count:     count,
	}

	return o
}

// repeated code in three of these functions; moved outside to make the code easier to read
func makeLengthMap(window common.Window) map[time.Time]float64 {
	bucketMap := window.BucketMap
	lengthMap := make(map[time.Time]float64)
	for timestamp, messages := range bucketMap {
		lengthMap[timestamp] = float64(len(messages))
	}
	return lengthMap
}

// Takes in map of time objects to frequencies and puts them into an ordered array of frequencies based on increasing timestamps
func GetSortedFrequencies(bucketMap map[time.Time]float64) ([]float64, []time.Time) {
	var timestamps []time.Time

	// Create a slice of timestamps and a corresponding slice of values in the order of timestamps
	for timestamp := range bucketMap {
		timestamps = append(timestamps, timestamp)
	}

	// Sort the timestamps in ascending order
	sort.Slice(timestamps, func(i, j int) bool {
		return timestamps[i].Before(timestamps[j])
	})

	// Create a new slice of values in the order of sorted timestamps
	sortedValues := make([]float64, len(timestamps))
	for i, timestamp := range timestamps {
		sortedValues[i] = bucketMap[timestamp]
	}

	return sortedValues, timestamps
}
