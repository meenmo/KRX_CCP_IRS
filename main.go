package main

import (
	"fmt"

	"github.com/meenmo/KRX_CCP_IRS/engine"
)

var ccpRates = map[float64]float64{
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
	trade := engine.Trade{
		EffectiveDate:   "2015-06-11",                              // 유효일
		TerminationDate: "2022-06-13",                              // 만기일
		FixedRate:       2.11250,                                   // 고정금리
		Notional:        10000000000,                               // 명목금액
		Position:        "rec",                                     // PAY, REC (거래 방향)
		Crv:             engine.BuildCurve("2021-12-20", ccpRates), // 결제일, CCP 금리 파라미터
	}
	fmt.Println(trade.NPV())
}
