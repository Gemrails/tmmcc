// RAINBOND, Application Management Platform
// Copyright (C) 2014-2017 Goodrain Co., Ltd.

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Rainbond,
// one or multiple Commercial Licenses authorized by Goodrain Co., Ltd.
// must be obtained first.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package metric

func calculate(timings *[TIMEBUCKETS]uint64) (fmin, favg, fmax float64) {
	var counts, total, min, max, avg uint64 = 0, 0, 0, 0, 0
	hasmin := false
	for _, val := range *timings {
		if val == 0 {
			// Queries should never take 0 nanoseconds. We are using 0 as a
			// trigger to mean 'uninitialized reading'.
			continue
		}
		if val < min || !hasmin {
			hasmin = true
			min = val
		}
		if val > max {
			max = val
		}
		counts++
		total += val
	}
	if counts > 0 {
		avg = total / counts // integer division
	}
	return float64(min) / 1000000, float64(avg) / 1000000,
		float64(max) / 1000000
}
