package irs

import (
	"math"
	"strings"
	"time"
)

type IRS struct {
	EffectiveDate   string  // 유효일
	TerminationDate string  // 만기일
	SettlementDate  string  // 결제일
	FixedRate       float64 // 고정금리
	Notional        float64 // 명목금액
	Position        string  // PAY, REC (거래 방향)
	CCPRates        map[float64]float64 // CCP 금리파라미터
}

func (irs IRS) cashFlow(Crv *Curve) (map[time.Time]float64, map[time.Time]float64) {
	// 고정금리레그 및 변동금리레그 현금흐름 산출
	var df float64
	var prevDf float64
	var floatingRate float64
	var pymtDate time.Time
	var prevPymtDate time.Time

	fixedCF := make(map[time.Time]float64)
	floatingCF := make(map[time.Time]float64)

	isFirstPayment := true
	effectiveDate   := dateParser(irs.EffectiveDate)
	terminationDate := dateParser(irs.TerminationDate)
	settlementDate  := dateParser(irs.SettlementDate)

	// Exceptions: 지급/수취 항목 입력 오류 (pay/rec)
	if !(strings.ToUpper(irs.Position) == "REC" || strings.ToUpper(irs.Position) == "PAY") {
		panic("Invalid Argument. The argument should be either 'REC' or 'PAY'")
	}

	for i := 0; modifiedFollowing(addMonth(effectiveDate, 3*i)).Before(terminationDate.AddDate(0, 0, 1)); i++ {
		// pymtDate(이자교환일)
		if isEOM(effectiveDate) {
			pymtDate = lastBusinessDayOfMonth(addMonth(effectiveDate, 3*i))
		} else {
			pymtDate = modifiedFollowing(addMonth(effectiveDate, 3*i))
		}
		// 유효일부터 3개월 단위로 이자교환일 넘어가면서, 이자교환일이 결제일 이후라면 PV 계산
		if pymtDate.After(settlementDate) {
			df = roundTo(math.Exp(-(days(settlementDate, pymtDate)/365)*(Crv.zeroRate(pymtDate)/100)), 12)

			if isFirstPayment {
				isFirstPayment = false
				prevPymtDate = funcPrevPymtDate(settlementDate, effectiveDate)
				floatingRate = cd91[priorBusinessDate(prevPymtDate).String()[:10]] / 100
			} else {
				floatingRate = ((prevDf / df) - 1) / (days(prevPymtDate, pymtDate) / 365)
			}

			// 고정/변동금리 레그의 각 기별 교환할 이자 산출
			fixedCF[pymtDate] = (irs.FixedRate / 100) * irs.Notional * days(prevPymtDate, pymtDate) / 365
			floatingCF[pymtDate] = (floatingRate) * irs.Notional * days(prevPymtDate, pymtDate) / 365

			// 다음 Loop에서 계산을 위해 해당 기의 할인계수 및 이자교환일 keep
			prevDf = df
			prevPymtDate = pymtDate
		}
	}
	return fixedCF, floatingCF
}

func (irs IRS) PVCashFlow(mapCashFlow map[time.Time]float64, Crv *Curve) map[time.Time]float64 {
	// 변동금리 현금흐름 map 혹은 고정금리 현금흐름 map을 받아 현재가치 map 산출
	var df float64

	for pymtDate, cf := range mapCashFlow {
		df = roundTo(math.Exp(-(days(dateParser(irs.SettlementDate), pymtDate)/365)*(Crv.zeroRate(pymtDate)/100)), 12)
		mapCashFlow[pymtDate] = df * cf
	}
	return mapCashFlow
}

func (irs IRS) PVByLeg(Crv *Curve) (float64, float64) {
	// 고정/변동 레그 별 현재가치 산출
	var sumFixedPV float64
	var sumFloatingPV float64

	fixedCF, floatingCF := irs.cashFlow(Crv)

	pvFixedLeg := irs.PVCashFlow(fixedCF, Crv)
	pvFloatingLeg := irs.PVCashFlow(floatingCF, Crv)

	for _, pv := range pvFixedLeg {
		sumFixedPV += pv
	}
	for _, pv := range pvFloatingLeg {
		sumFloatingPV += pv
	}
	return sumFixedPV, sumFloatingPV
}

func (irs IRS) NPV(Crv *Curve) float64 {
	// NPV 산출
	sumFixedPV, sumFloatingPV := irs.PVByLeg(Crv)

	if strings.ToUpper(irs.Position) == "REC" {
		return sumFixedPV - sumFloatingPV
	} else {
		return sumFloatingPV - sumFixedPV
	}
}