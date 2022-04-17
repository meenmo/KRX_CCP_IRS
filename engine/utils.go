package irs

import (
	"log"
	"math"
	"time"
)

func roundTo(n float64, decimals uint32) float64 {
	// 반올림
	return math.Round(n*math.Pow(10, float64(decimals))) / math.Pow(10, float64(decimals))
}

func days(date1 time.Time, date2 time.Time) float64 {
	// 날짜 간 일수 계산
	return date2.Sub(date1).Hours() / 24
}

func isEOM(effectiveDate time.Time) bool {
	// EOM(End of Month rule) 해당하는지 체크
	if effectiveDate == lastBusinessDayOfMonth(effectiveDate) {
		return true
	} else {
		return false
	}
}

func isHoliday(date time.Time) bool {
	// 해당 일자가 공휴일인지 체크
	for _, holiday := range arrHolidays {
		if holiday == date.String()[:10] {
			return true
		}
	}
	return false
}

func dateParser(strDate string) time.Time {
	// string "YYYY-MM-DD" -> time.Time
	const layout = "2006-01-02"
	date, err := time.Parse(layout, strDate)
	if err != nil {
		log.Fatal(err)
	}
	return date
}

func monthInt(date time.Time) int {
	// time.Time -> month (int)
	return int(date.Month())
}

func addMonth(date time.Time, month int) time.Time {
	// Go time package의 AddDate 버그로 인한 월간 날짜 이동 Helper Function (equivalent to 'EDATE' in excel)
	// 엑셀이나 다른 프로그램의 날짜함수와 달리, Go time package에서 날짜를 조정하는 경우 Normalize를 하는데,
	// 예를 들면, AddDate를 이용하여 3월 30일에서 1달을 빼면 2월 28일이 아닌 3월 2일인 식이다
	// 이와 같은 버그를 보정한 Helper Function
	// 버그 관련 사항은 https://github.com/golang/go/issues/31145 참고

	adjDate := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, month, 0) // 해당 월의 1일로 초기화
	if adjDate.Month() == date.AddDate(0, month, 0).Month() {
		// AddDate를 하여 정상적인 결과가 나오는 경우
		// (해당 월 1일자로 AddDate를 했을 때와 해당 일 기준으로 AddDate를 했을 때와 동일한 월인 경우, 즉 이상이 없는 경우)
		return date.AddDate(0, month, 0)
	} else {
		// AddDate 값이 에러인 경우 (보통 월말 부근에서 AddDate를 하는 경우 에러 발생)
		// 전월 말까지 일수 차감
		date = date.AddDate(0, month, 0)
		month := monthInt(date)
		for monthInt(date) == month {
			date = date.AddDate(0, 0, -1)
		}
		return date
	}
}

func funcPrevPymtDate(settlementDate time.Time, effectiveDate time.Time) time.Time {
	// 직전 이자지급일 산출 함수

	var date time.Time

	// effectiveDate := dateParser(irs.effectiveDate)
	// settlementDate := dateParser(irs.settlementDate)

	for i := 0; modifiedFollowing(addMonth(effectiveDate, 3*i)).Before(settlementDate.AddDate(0, 0, 1)); i++ {
		date = modifiedFollowing(addMonth(effectiveDate, 3*i))
	}

	if isEOM(effectiveDate) {
		return lastBusinessDayOfMonth(date)
	} else {
		return date
	}
}

func priorBusinessDate(date time.Time) time.Time {
	// 직전 영업일 산출 함수
	date = date.AddDate(0, 0, -1)
	for date.Weekday().String() == "Saturday" || date.Weekday().String() == "Sunday" || isHoliday(date) == true {
		date = date.AddDate(0, 0, -1)
	}
	return date
}

func modifiedFollowing(date time.Time) time.Time {
	// Modified Following Rule에 해당하는 일자 반환
	month := monthInt(date)
	for date.Weekday().String() == "Saturday" || date.Weekday().String() == "Sunday" || isHoliday(date) == true {
		date = date.AddDate(0, 0, 1)
	}
	for month < monthInt(date) {
		date = date.AddDate(0, 0, -1)
		for date.Weekday().String() == "Saturday" || date.Weekday().String() == "Sunday" || isHoliday(date) == true {
			date = date.AddDate(0, 0, -1)
		}
	}
	return date
}

func lastBusinessDayOfMonth(date time.Time) time.Time {
	// 해당 월의 마지막 영업일 반환
	var nextMonth int
	if monthInt(date) == 12 {
		nextMonth = 1
	} else {
		nextMonth = monthInt(date) + 1
	}
	for !(monthInt(date) == nextMonth) {
		date = date.AddDate(0, 0, 1)
	}
	return priorBusinessDate(date)
}

func adjacentDates(givenDate time.Time, pymtDates []time.Time) (time.Time, time.Time) {
	// 유효일부터 3개월 단위 커브에서 해당 날짜와 인접한 두 날짜를 반환
	date1 := pymtDates[0]
	date2 := pymtDates[1]

	for _, date := range pymtDates[2:] {
		if date1.Before(givenDate) && givenDate.Before(date2) {
			return date1, date2
		}
		date1 = date2
		date2 = date
	}
	return date1, date2
}

func adjacentDatesCCPCurve(givenDate time.Time, pymtDates []time.Time, ccpRatesParams map[float64]float64) (time.Time, time.Time) {
	// KRX에서 제공하는 금리파라미터의 기간(1일, 91일,...,20년)에서 해당 날짜와 인접한 두 날짜를 반환
	date1 := pymtDates[0]
	date2 := pymtDates[1]

	date2Term := pymtDatesToFloat(pymtDates)
	for _, date := range pymtDates[2:] {
		if date1.Before(givenDate) && givenDate.Before(date2) {
			return date1, date2
		}
		term := date2Term[date]
		if _, _bool := ccpRatesParams[term]; _bool {
			date1 = date2
			date2 = date
		}
	}
	return date1, date2
}

func pymtDatesToFloat(arrDates []time.Time) map[time.Time]float64 {
	// 가상스왑의 이자교환일을 연 기준 숫자형으로 변환
	// date -> 0, 0.25, 0.5, 0.75, 1, 1.25, ...
	_map := make(map[time.Time]float64)
	pymtDate := arrDates[0]
	terminationDate := arrDates[len(arrDates)-1].AddDate(0, 0, 1)

	for i := 0.0; pymtDate.Before(terminationDate); i++ {
		_map[modifiedFollowing(pymtDate)] = i * 0.25
		pymtDate = pymtDate.AddDate(0, 3, 0)
	}

	return _map
}
