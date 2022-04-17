package main

import (
	"fmt"

	"github.com/meenmo/KRX_CCP_IRS/engine"	
)

var ccpRates211217 = map[float64]float64{
	// CCP 금리 파라미터
	0:    0.9880102510,
	0.25: 1.2700000000,
	0.5:  1.3764285714,
	0.75: 1.4614285714,
	1:    1.5514285714,
	1.5:  1.6739285714,
	2:    1.7389285714,
	3:    1.7914285714,
	4:    1.8075000000,
	5:    1.8050000000,
	6:    1.8003571429,
	7:    1.7875000000,
	8:    1.7842857143,
	9:    1.7867857143,
	10:   1.7875000000,
	12:   1.7792857143,
	15:   1.6760714286,
	20:   1.5525000000,
}

func main() {
	trade := irs.IRS{
		SettlementDate:  "2021-12-20",   // 결제일
		EffectiveDate:   "2016-12-19",   // 유효일
		TerminationDate: "2026-12-21",   // 만기일
		FixedRate:       1.9275,         // 고정금리
		Notional:        15000000000,    // 명목금액
		Position:        "pay",          // PAY, REC (거래 방향)
		CCPRates:        ccpRates211217, // CCP 금리파라미터
	}

	Crv := irs.BuildCurve(trade.SettlementDate, trade.CCPRates)
	fmt.Println(int(trade.NPV(Crv))) // NPV
	
	fmt.Println(trade.Delta(5)) 	 // 5 Key Rate Delta
}
