package engine

import (
	"math"
	"time"
)

type Curve struct {
	settlementDate time.Time             // 결제일
	ccpRates       map[float64]float64   // CCP 금리 파라미터
	arrPymtDates   []time.Time           // 이자지급일
	swap           map[time.Time]float64 // 가상스왑 금리커브
	df             map[time.Time]float64 // 가상스왑 금리커브 할인계수
	zero           map[time.Time]float64 // Zero Curve
}

func BuildCurve(settlementDate string, ccpRates map[float64]float64) *Curve {
	// struct Curve 생성 함수: 프라이싱에 필요한 일별 데이터 생성
	crv := new(Curve)
	crv.settlementDate = dateParser(settlementDate)
	crv.ccpRates = ccpRates
	crv.arrPymtDates = crv.pymtDates()
	crv.swap = crv.swapCurve()
	crv.df = crv.discountFactors()
	crv.zero = crv.zeroCurve()
	return crv
}

func (crv Curve) pymtDates() []time.Time {
	// 가상스왑의 미래 이자교환일 산출 함수 (결제일 기준; 이자교환주기 3개월)
	var pymtDate time.Time
	arrDates := make([]time.Time, 0)
	for i := 0; i <= 80; i++ {
		// Over 20 years
		pymtDate = crv.settlementDate.AddDate(0, 3*i, 0)
		arrDates = append(arrDates, modifiedFollowing(pymtDate))
	}
	return arrDates
}

func (crv Curve) swapCurve() map[time.Time]float64 {
	// 가상스왑 금리커브 산출 함수 (input swap rates 보간하여 커브 생성)
	sc := make(map[time.Time]float64)
	arrPymtDates := crv.pymtDates()
	date2Term := pymtDatesToFloat(arrPymtDates)

	for _, date := range arrPymtDates {
		term := date2Term[date]
		if rate, isKeyTenors := crv.ccpRates[term]; isKeyTenors {
			sc[date] = (rate) / 100
		} else {
			date1, date2 := adjacentDatesCCPCurve(date, arrPymtDates, crv.ccpRates)
			rate1 := crv.ccpRates[date2Term[date1]]
			rate2 := crv.ccpRates[date2Term[date2]]
			sc[date] = (rate1 + (rate2-rate1)*days(date1, date)/days(date1, date2)) / 100
		}
	}
	return sc
}

func (crv Curve) discountFactors() map[time.Time]float64 {
	// 가상스왑 금리커브 -> 할인계수 산출
	df := make(map[time.Time]float64)

	swapCurve := crv.swap
	arrPymtDates := crv.arrPymtDates

	prevDate := crv.arrPymtDates[0]
	numerator := 0.0

	df[prevDate] = 1 // Discount Factor is 1 initially
	for i, date1 := range arrPymtDates[1:] {
		swapRate := swapCurve[date1]
		if i == 0 {
			numerator = 1
		} else {
			prevDate2 := arrPymtDates[0]
			for _, date2 := range arrPymtDates[1 : i+1] {
				numerator += days(prevDate2, date2) * df[date2]
				prevDate2 = date2
			}
			numerator = 1 - (numerator/365)*swapRate
		}

		df[date1] = roundTo(numerator/(1+swapRate*days(prevDate, date1)/365), 12)
		prevDate = date1
		numerator = 0
	}
	return df
}

func (crv Curve) zeroCurve() map[time.Time]float64 {
	// 가상스왑의 할인계수 -> Zero Curve 생성
	zc := make(map[time.Time]float64)
	for i, date := range crv.arrPymtDates {
		if i == 0 {
			zc[date] = roundTo(crv.swap[date]*100, 12)
		} else {
			df := crv.df[date]
			dayCountFrac := days(crv.settlementDate, date) / 365
			zc[date] = roundTo(-math.Log(df)/dayCountFrac*100, 12)
		}
	}
	return zc
}

func (crv Curve) zeroRate(pymtDate time.Time) float64 {
	// 위에서 산출한 Zero Curve에서 선형보간한 무이표금리
	if _zeroRate, isInArray := crv.zero[pymtDate]; isInArray {
		return _zeroRate
	} else {
		date1, date2 := adjacentDates(pymtDate, crv.arrPymtDates)
		rate1 := crv.zero[date1]
		rate2 := crv.zero[date2]
		return roundTo(rate1+(rate2-rate1)*days(date1, pymtDate)/days(date1, date2), 12)
	}
}
